package handlers

import (
	"log"
	"strconv"
	"sync"

	"action_users/constants"
	"action_users/controller"
	"action_users/models"
	"action_users/repositories"

	"github.com/gofiber/fiber/v2"
)

// Handler управляет HTTP-запросами.
type Handler struct {
	ctrl *controller.Controller
}

// NewHandler создает новый экземпляр Handler.
func NewHandler(ctrl *controller.Controller) *Handler {
	return &Handler{ctrl: ctrl}
}

// ProcessUsers обрабатывает запрос на получение данных пользователей.
func (h *Handler) ProcessUsers(c *fiber.Ctx) error {
	// Получаем параметры из запроса
	monthsStr := c.Query("months", "1")
	countryIdStr := c.Query("countryId", "0")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "100")

	// Парсим параметры
	months, err := strconv.Atoi(monthsStr)
	if err != nil || months < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid months parameter",
		})
	}

	countryId, err := strconv.Atoi(countryIdStr)
	if err != nil || countryId < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid countryId parameter",
		})
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid page parameter",
		})
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid limit parameter (max 1000)",
		})
	}

	// Вычисляем from
	from := (page - 1) * limit

	// Получаем userIds - ИСПРАВЛЕНО: используем h.ctrl.Client()
	userIds, err := repositories.GetUserIds(h.ctrl.Client(), from, limit, countryId)
	if err != nil {
		log.Printf("error: getUserIds failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch user IDs",
		})
	}

	if len(userIds) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "no users found",
			"summary": fiber.Map{
				"orphanUsersCount":         0,
				"inactiveUsersCount":       0,
				"registeredNoActionsCount": 0,
			},
		})
	}

	// Обработка пользователей
	var mu sync.Mutex
	var wg sync.WaitGroup
	var inactiveResults []models.ClientData
	var registeredNoActions []models.ClientData
	var orphanUsers []models.ClientData

	for _, uid := range userIds {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()

			actions, err := h.ctrl.GetLastTwoActionsForUser(uid, countryId)
			if err != nil {
				log.Printf("warn: getLastTwoActionsForUser(%s) error: %v", uid, err)
				return
			}

			if len(actions) == 0 {
				// ИСПРАВЛЕНО: используем h.ctrl.Client()
				clientHit, err := repositories.GetClientById(h.ctrl.Client(), uid, countryId)
				if err != nil {
					log.Printf("warn: getClientById(%s) error: %v", uid, err)
					return
				}
				if clientHit == nil {
					log.Printf("warn: registered user not found AND no actions for userId: %s", uid)
					return
				}
				cd := h.ctrl.BuildClientData(clientHit, nil, nil, nil, countryId, uid)
				mu.Lock()
				registeredNoActions = append(registeredNoActions, cd)
				mu.Unlock()
				return
			}

			if h.ctrl.CheckUserActionsInterval(actions, months) {
				// ИСПРАВЛЕНО: используем h.ctrl.Client()
				clientHit, err := repositories.GetClientById(h.ctrl.Client(), uid, countryId)
				if err != nil {
					log.Printf("warn: getClientById(%s) error: %v", uid, err)
					return
				}

				topUpSrc, _ := h.ctrl.GetLastActionFromIndices(uid, constants.TopUpIndices, countryId)
				betSrc, _ := h.ctrl.GetLastActionFromIndices(uid, constants.BetIndices, countryId)
				withdrawalSrc, _ := h.ctrl.GetLastActionFromIndices(uid, constants.WithdrawalIndices, countryId)

				if clientHit == nil {
					log.Printf("info: orphan user %s - no client data but has actions", uid)
					cd := h.ctrl.BuildClientData(nil, topUpSrc, betSrc, withdrawalSrc, countryId, uid)
					mu.Lock()
					orphanUsers = append(orphanUsers, cd)
					mu.Unlock()
					return
				}

				cd := h.ctrl.BuildClientData(clientHit, topUpSrc, betSrc, withdrawalSrc, countryId, uid)
				mu.Lock()
				inactiveResults = append(inactiveResults, cd)
				mu.Unlock()
			}
		}(uid)
	}
	wg.Wait()

	// Формируем ответ
	response := fiber.Map{
		"orphanUsers":         orphanUsers,
		"inactiveUsers":       inactiveResults,
		"registeredNoActions": registeredNoActions,
		"summary": fiber.Map{
			"orphanUsersCount":   len(orphanUsers),
			"inactiveUsersCount": len(inactiveResults),
			//"registeredNoActionsCount": len(registeredNoActions),
			"totalProcessed": len(userIds),
			"page":           page,
			"limit":          limit,
			"months":         months,
			"countryId":      countryId,
		},
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// HealthCheck для проверки состояния сервера
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "Server is running",
	})
}
