package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/AlexBlackNn/authloyalty/sso/internal/dto"

	"github.com/AlexBlackNn/authloyalty/sso/internal/config"
	"github.com/AlexBlackNn/authloyalty/sso/internal/services/authservice"
	"github.com/AlexBlackNn/authloyalty/sso/pkg/storage"
	"github.com/go-playground/validator/v10"
)

type httpAuthorization interface {
	Login(
		ctx context.Context,
		reqData *dto.Login,
	) (accessToken string, refreshToken string, err error)
	Register(
		ctx context.Context,
		reqData *dto.Register,
	) (ctxOut context.Context, userID string, err error)
	Logout(
		ctx context.Context,
		reqData *dto.Logout,
	) (success bool, err error)
	Refresh(
		ctx context.Context,
		reqData *dto.Refresh,
	) (accessToken string, refreshToken string, err error)
}

type AuthHandlers struct {
	log  *slog.Logger
	auth httpAuthorization
	cfg  *config.Config
}

func New(log *slog.Logger, cfg *config.Config, authService httpAuthorization) AuthHandlers {
	return AuthHandlers{log: log, cfg: cfg, auth: authService}
}

// handleBadRequest validates post body and writes messages to client. In case of using "err := render.DecodeJSON(r.Body, &reqData)"
// can be written as handleBadRequest[T Login | Logout | Refresh | Register](...). In case of using easyjson Login,
// Logout, Refresh, Register have  UnmarshalJSON method after code generation. json.Unmarshaler must be use here to
// work with easyjson.
// Unmarshaler provides ability easyjson lib to work with generic type.
// In case of using "err := render.DecodeJSON(r.Body, &reqData)" it can be deleted.
func handleBadRequest[T json.Unmarshaler](w http.ResponseWriter, r *http.Request, reqData T) (T, error) {
	if r.Method != http.MethodPost {
		dto.ResponseErrorNowAllowed(w, "only POST method allowed")
		return reqData, errors.New("method not allowed")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		dto.ResponseErrorBadRequest(w, "failed to read body")
		return reqData, errors.New("failed to read body")
	}
	err = reqData.UnmarshalJSON(body)
	if err != nil {
		dto.ResponseErrorBadRequest(w, "failed to decode request")
		return reqData, errors.New("failed to decode request")
	}
	if err = validator.New().Struct(reqData); err != nil {
		var validateErr validator.ValidationErrors
		if errors.As(err, &validateErr) {
			dto.ResponseErrorBadRequest(w, dto.ValidationError(validateErr))
			return reqData, errors.New("validation error")
		}
		dto.ResponseErrorBadRequest(w, "bad request")
		return reqData, errors.New("bad request")
	}
	return reqData, nil
}

func ctxWithTimeoutCause(r *http.Request, cfg *config.Config, textError string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(
		r.Context(),
		time.Duration(cfg.ServerHandlersTimeouts.LoginTimeoutMs)*time.Millisecond,
		errors.New(textError),
	)
	return ctx, cancel
}

// @Summary Login
// @Description Authenticates a user and returns access and refresh tokens.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.Login true "Login request"
// @Success 201 {object} models.Response "Login successful"
// @Router /auth/login [post]
func (a *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {

	reqData, err := handleBadRequest[*dto.Login](w, r, &dto.Login{})
	if err != nil {
		return
	}

	ctx, cancel := ctxWithTimeoutCause(r, a.cfg, "login timeout")
	defer cancel()

	accessToken, refreshToken, err := a.auth.Login(ctx, reqData)
	if err != nil {
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			dto.ResponseErrorNotFound(w, "user not found")
			return
		}
		dto.ResponseErrorInternal(w, "internal server error")
		return
	}
	dto.ResponseOKAccessRefresh(w, accessToken, refreshToken)
}

// @Summary Logout
// @Description Logout from current session. Frontend needs to send access and then refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.Logout true "Logout request"
// @Success 200 {object} models.Response "Logout successful"
// @Router /auth/logout [post]
func (a *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	reqData, err := handleBadRequest[*dto.Logout](w, r, &dto.Logout{})
	if err != nil {
		return
	}
	ctx, cancel := ctxWithTimeoutCause(r, a.cfg, "logout timeout")
	defer cancel()

	_, err = a.auth.Logout(ctx, reqData)
	if err != nil {
		switch {
		case errors.Is(err, authservice.ErrUserNotFound):
			dto.ResponseErrorNotFound(w, "user not found")
		case errors.Is(err, authservice.ErrTokenRevoked):
			dto.ResponseErrorStatusConflict(w, "token revoked")
		case errors.Is(err, authservice.ErrTokenParsing):
			dto.ResponseErrorBadRequest(w, "token error")
		case errors.Is(err, authservice.ErrTokenTTLExpired):
			dto.ResponseErrorStatusConflict(w, "token ttl expired")
		default:
			dto.ResponseErrorInternal(w, "internal server error")
		}
		return
	}
	dto.ResponseOK(w)
}

// @Summary Registration
// @Description User registration
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.Register true "Register request"
// @Success 201 {object} models.Response "Register successful"
// @Router /auth/registration [post]
func (a *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	reqData, err := handleBadRequest[*dto.Register](w, r, &dto.Register{})
	if err != nil {
		return
	}
	ctx, cancel := ctxWithTimeoutCause(r, a.cfg, "register timeout")
	defer cancel()

	ctx, _, err = a.auth.Register(ctx, reqData)
	if err != nil {
		// TODO change to service error
		if errors.Is(err, storage.ErrUserExists) {
			dto.ResponseErrorStatusConflict(w, "user already exists")
			return
		}
		dto.ResponseErrorInternal(w, "internal server error")
		return
	}

	accessToken, refreshToken, err := a.auth.Login(
		ctx, &dto.Login{Email: reqData.Email, Password: reqData.Password},
	)

	if err != nil {
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			dto.ResponseErrorNotFound(w, "user not found")
			return
		}
		dto.ResponseErrorInternal(w, "internal server error")
		return
	}
	dto.ResponseOKAccessRefresh(w, accessToken, refreshToken)
}

// @Summary Refresh
// @Description
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.Refresh true "Refresh request"
// @Success 201 {object} models.Response "Refresh successful"
// @Router /auth/refresh [post]
func (a *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	reqData, err := handleBadRequest[*dto.Refresh](w, r, &dto.Refresh{})
	if err != nil {
		return
	}

	ctx, cancel := ctxWithTimeoutCause(r, a.cfg, "refresh timeout")
	defer cancel()

	accessToken, refreshToken, err := a.auth.Refresh(ctx, reqData)

	if err != nil {
		switch {
		case errors.Is(err, authservice.ErrUserNotFound):
			dto.ResponseErrorNotFound(w, "user not found")
		case errors.Is(err, authservice.ErrTokenWrongType):
			dto.ResponseErrorStatusConflict(w, "token wrong type, expected refresh")
		case errors.Is(err, authservice.ErrTokenRevoked):
			dto.ResponseErrorStatusConflict(w, "token revoked")
		case errors.Is(err, authservice.ErrTokenParsing):
			dto.ResponseErrorBadRequest(w, "token error")
		case errors.Is(err, authservice.ErrTokenTTLExpired):
			dto.ResponseErrorStatusConflict(w, "token ttl expired")
		default:
			dto.ResponseErrorInternal(w, "internal server error")
		}
		return
	}
	dto.ResponseOKAccessRefresh(w, accessToken, refreshToken)
}
