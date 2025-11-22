package subscription

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	layoutYearMonth     = "2006-01"
	layoutMonthYear     = "01-2006"
	layoutFullDate      = "2006-01-02"
	defaultDayComponent = 1
	defaultPage         = 1
	defaultLimit        = 2
	maxLimit            = 100
)

// Handler exposes HTTP handlers for subscription resources.
type Handler struct {
	svc    Service
	logger *slog.Logger
}

type errorResponse struct {
	Error string `json:"error"`
}

type summaryResponse struct {
	TotalPrice int `json:"total_price"`
}

type listResponse struct {
	Items []Subscription `json:"items"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int            `json:"total"`
}

func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{svc: service, logger: logger}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/subscriptions")
	group.POST("", h.create)
	group.GET("", h.list)
	group.GET("/summary", h.summary)
	group.GET("/:id", h.getByID)
	group.PATCH("/:id", h.update)
	group.DELETE("/:id", h.delete)
}

type createSubscriptionRequest struct {
	ServiceName string  `json:"service_name" binding:"required"`
	PriceRUB    int     `json:"price" binding:"required,min=0"`
	UserID      string  `json:"user_id" binding:"required"`
	StartMonth  string  `json:"start_date" binding:"required"`
	EndMonth    *string `json:"end_date"`
}

// create godoc
// @Summary Create subscription
// @Description Create a new subscription entry
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body createSubscriptionRequest true "Subscription payload"
// @Success 201 {object} Subscription
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions [post]
func (h *Handler) create(c *gin.Context) {
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Info("invalid create payload", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.logger.Info("invalid user id", "user_id", req.UserID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	startMonth, err := parseMonth(req.StartMonth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var end *time.Time
	if req.EndMonth != nil && strings.TrimSpace(*req.EndMonth) != "" {
		parsed, err := parseMonth(*req.EndMonth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if parsed.Before(startMonth) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_date cannot be before start_date"})
			return
		}
		end = &parsed
	}

	sub, err := h.svc.Create(c.Request.Context(), CreateParams{
		ServiceName: strings.TrimSpace(req.ServiceName),
		PriceRUB:    req.PriceRUB,
		UserID:      userID,
		StartMonth:  startMonth,
		EndMonth:    end,
	})
	if err != nil {
		h.logger.Error("failed to create subscription", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// list godoc
// @Summary List subscriptions
// @Description List subscriptions ordered by creation date with pagination
// @Tags subscriptions
// @Produce json
// @Param page query int false "Page number (>=1)" default(1)
// @Param limit query int false "Items per page (<=100)" default(20)
// @Success 200 {object} listResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions [get]
func (h *Handler) list(c *gin.Context) {
	page := parsePositiveInt(c.DefaultQuery("page", "1"), defaultPage)
	limit := parsePositiveInt(c.DefaultQuery("limit", fmt.Sprintf("%d", defaultLimit)), defaultLimit)
	if limit > maxLimit {
		limit = maxLimit
	}

	opts := ListOptions{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	subs, total, err := h.svc.List(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list subscriptions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, listResponse{
		Items: subs,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

// getByID godoc
// @Summary Get subscription
// @Description Get subscription by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 200 {object} Subscription
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions/{id} [get]
func (h *Handler) getByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.logger.Info("invalid subscription id", "id", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	sub, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		// Previously compared using == which fails for wrapped errors.
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Info("subscription not found", "id", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.logger.Error("failed to get subscription", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

type updateSubscriptionRequest struct {
	ServiceName *string `json:"service_name"`
	PriceRUB    *int    `json:"price"`
	StartMonth  *string `json:"start_date"`
	EndMonth    *string `json:"end_date"`
}

// update godoc
// @Summary Update subscription
// @Description Partially update subscription fields
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Subscription ID"
// @Param request body updateSubscriptionRequest true "Fields to update"
// @Success 200 {object} Subscription
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions/{id} [patch]
func (h *Handler) update(c *gin.Context) {
	idParam := c.Param("id")
	subID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req updateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Info("invalid update payload", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := UpdateParams{ID: subID}

	if req.ServiceName != nil {
		trimmed := strings.TrimSpace(*req.ServiceName)
		params.ServiceName = &trimmed
	}

	if req.PriceRUB != nil {
		if *req.PriceRUB < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price cannot be negative"})
			return
		}
		params.PriceRUB = req.PriceRUB
	}

	if req.StartMonth != nil {
		start, err := parseMonth(*req.StartMonth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		params.StartMonth = &start
	}

	if req.EndMonth != nil {
		params.EndMonthSet = true
		if strings.TrimSpace(*req.EndMonth) == "" {
			params.EndMonth = nil
		} else {
			end, err := parseMonth(*req.EndMonth)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if params.StartMonth != nil && end.Before(*params.StartMonth) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "end_date cannot be before start_date"})
				return
			}
			params.EndMonth = &end
		}
	}

	sub, err := h.svc.Update(c.Request.Context(), params)
	if err != nil {
		// Previously compared using == which fails for wrapped errors.
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Info("subscription not found for update", "id", idParam)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.logger.Error("failed to update subscription", "id", idParam, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// delete godoc
// @Summary Delete subscription
// @Description Delete subscription by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 204 {string} string "No Content"
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.logger.Info("invalid subscription id for delete", "id", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		// Previously compared using == which fails for wrapped errors.
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Info("subscription not found for delete", "id", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.logger.Error("failed to delete subscription", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// summary godoc
// @Summary Sum subscriptions
// @Description Calculate total subscription cost within optional filters
// @Tags subscriptions
// @Produce json
// @Param start query string false "Start month (YYYY-MM or MM-YYYY)"
// @Param end query string false "End month (YYYY-MM or MM-YYYY)"
// @Param user_id query string false "User ID (UUID)"
// @Param service_name query string false "Service name"
// @Success 200 {object} summaryResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /subscriptions/summary [get]
func (h *Handler) summary(c *gin.Context) {
	var (
		startMonth *time.Time
		endMonth   *time.Time
		userID     *uuid.UUID
		service    *string
		err        error
	)

	if start := c.Query("start"); start != "" {
		if startMonth, err = parseMonthPtr(start); err != nil {
			h.logger.Info("invalid start date", "value", start)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if end := c.Query("end"); end != "" {
		if endMonth, err = parseMonthPtr(end); err != nil {
			h.logger.Info("invalid end date", "value", end)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if startMonth != nil && endMonth != nil && endMonth.Before(*startMonth) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end must be after start"})
		return
	}

	if user := c.Query("user_id"); user != "" {
		parsed, err := uuid.Parse(user)
		if err != nil {
			h.logger.Info("invalid user_id filter", "user_id", user)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userID = &parsed
	}

	if name := strings.TrimSpace(c.Query("service_name")); name != "" {
		service = &name
	}

	total, err := h.svc.SumByPeriod(c.Request.Context(), SumFilter{
		StartMonth:  startMonth,
		EndMonth:    endMonth,
		UserID:      userID,
		ServiceName: service,
	})
	if err != nil {
		h.logger.Error("failed to summarize subscriptions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_price": total})
}

func parseMonth(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("date value cannot be empty")
	}

	if t, err := time.Parse(layoutYearMonth, value); err == nil {
		return normalizeMonth(t), nil
	}
	if t, err := time.Parse(layoutMonthYear, value); err == nil {
		return normalizeMonth(t), nil
	}
	// Allow full date inputs and truncate.
	if t, err := time.Parse(layoutFullDate, value); err == nil {
		return normalizeMonth(t), nil
	}

	return time.Time{}, fmt.Errorf("date must be in YYYY-MM or MM-YYYY format")
}

func parseMonthPtr(value string) (*time.Time, error) {
	t, err := parseMonth(value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func normalizeMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), defaultDayComponent, 0, 0, 0, 0, time.UTC)
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
