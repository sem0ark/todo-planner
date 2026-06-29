package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db *pgxpool.Pool
}

func NewAPI(db *pgxpool.Pool) *API {
	return &API{db: db}
}

func NewCORSMiddleware(allowedOrigins []string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			isAllowed := false
			for _, allowed := range allowedOrigins {
				if allowed == "*" || origin == allowed {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				w.Header().Add("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}
}

func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := api.db.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (api *API) TodosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		api.getTodos(w, r)
	case http.MethodPost:
		api.createTodo(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *API) TodoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.URL.Path[len("/todos/"):])
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.getTodo(w, r, id)
	case http.MethodPut:
		api.updateTodo(w, r, id)
	case http.MethodDelete:
		api.deleteTodo(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *API) getTodos(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, description, completed, priority, due_date, created_at, updated_at
		FROM todos
		ORDER BY
			CASE priority
				WHEN 'high' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 3
				ELSE 4
			END,
			created_at DESC
	`

	rows, err := api.db.Query(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&todo.Completed,
			&todo.Priority,
			&todo.DueDate,
			&todo.CreatedAt,
			&todo.UpdatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	if todos == nil {
		todos = []Todo{}
	}

	json.NewEncoder(w).Encode(todos)
}

func (api *API) getTodo(w http.ResponseWriter, r *http.Request, id int) {
	var todo Todo
	query := `
		SELECT id, title, description, completed, priority, due_date, created_at, updated_at
		FROM todos
		WHERE id = $1
	`

	err := api.db.QueryRow(r.Context(), query, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Completed,
		&todo.Priority,
		&todo.DueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todo)
}

func (api *API) createTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if todo.Priority == "" {
		todo.Priority = "medium"
	}

	if err := validateTodoInput(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO todos (title, description, completed, priority, due_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := api.db.QueryRow(r.Context(), query,
		todo.Title,
		todo.Description,
		todo.Completed,
		todo.Priority,
		todo.DueDate,
	).Scan(&todo.ID, &todo.CreatedAt, &todo.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

func (api *API) updateTodo(w http.ResponseWriter, r *http.Request, id int) {
	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateTodoInput(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE todos
		SET title = $1, description = $2, completed = $3, priority = $4, due_date = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING updated_at
	`

	err := api.db.QueryRow(r.Context(), query,
		todo.Title,
		todo.Description,
		todo.Completed,
		todo.Priority,
		todo.DueDate,
		id,
	).Scan(&todo.UpdatedAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	todo.ID = id
	json.NewEncoder(w).Encode(todo)
}

func (api *API) deleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	result, err := api.db.Exec(r.Context(), "DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateTodoInput(todo *Todo) error {
	if todo.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(todo.Title) > 255 {
		return fmt.Errorf("title must not exceed 255 characters")
	}
	if len(todo.Description) > 2000 {
		return fmt.Errorf("description must not exceed 2000 characters")
	}
	if todo.Priority != "low" && todo.Priority != "medium" && todo.Priority != "high" {
		return fmt.Errorf("priority must be 'low', 'medium', or 'high'")
	}
	return nil
}
