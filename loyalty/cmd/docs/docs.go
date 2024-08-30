// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "API Support"
        },
        "license": {
            "name": "Apache 2.0"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/auth/healthz": {
            "get": {
                "description": "Определяет, нужно ли перезагрузить сервис",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Health"
                ],
                "summary": "Проверка, что приложение живо",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/dto.Response"
                        }
                    }
                }
            }
        },
        "/auth/ready": {
            "get": {
                "description": "Определяет можно ли подавать трафик на сервис",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Health"
                ],
                "summary": "Проверка готовности приложения",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/dto.Response"
                        }
                    }
                }
            }
        },
        "/loyalty": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Authenticates a user and returns access and refresh tokens.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loyalty"
                ],
                "summary": "AddLoyalty",
                "parameters": [
                    {
                        "description": "UserLoyalty request",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.UserLoyalty"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Get loyalty successful",
                        "schema": {
                            "$ref": "#/definitions/dto.Response"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Authenticates a user and returns access and refresh tokens.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loyalty"
                ],
                "summary": "AddLoyalty",
                "parameters": [
                    {
                        "description": "UserLoyalty request",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.UserLoyalty"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Add loyalty successful",
                        "schema": {
                            "$ref": "#/definitions/dto.Response"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "dto.Response": {
            "type": "object",
            "properties": {
                "balance": {
                    "type": "integer"
                },
                "error": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "dto.UserLoyalty": {
            "type": "object",
            "required": [
                "balance",
                "comment",
                "operation"
            ],
            "properties": {
                "balance": {
                    "type": "integer"
                },
                "comment": {
                    "type": "string"
                },
                "operation": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Loyalty",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8001",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Swagger API",
	Description:      "loyalty service.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
