package handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"statusboard/internal/checker"
	"statusboard/internal/database"
)

type Handler struct {
	db        *database.DB
	templates *template.Template
}

