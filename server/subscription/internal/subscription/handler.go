package subscription

import (
	"database/sql"
	"fmt"
	"net/http"
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
)

// Handler exposes HTTP handlers for subscription resources.
type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
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

func (h *Handler) create(c *gin.Context) {
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
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

	sub, err := h.repo.Create(c.Request.Context(), CreateParams{
		ServiceName: strings.TrimSpace(req.ServiceName),
		PriceRUB:    req.PriceRUB,
		UserID:      userID,
		StartMonth:  startMonth,
		EndMonth:    end,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) list(c *gin.Context) {
	subs, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *Handler) getByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	sub, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
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

func (h *Handler) update(c *gin.Context) {
	idParam := c.Param("id")
	subID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req updateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	sub, err := h.repo.Update(c.Request.Context(), params)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if end := c.Query("end"); end != "" {
		if endMonth, err = parseMonthPtr(end); err != nil {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userID = &parsed
	}

	if name := strings.TrimSpace(c.Query("service_name")); name != "" {
		service = &name
	}

	total, err := h.repo.SumByPeriod(c.Request.Context(), SumFilter{
		StartMonth:  startMonth,
		EndMonth:    endMonth,
		UserID:      userID,
		ServiceName: service,
	})
	if err != nil {
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
