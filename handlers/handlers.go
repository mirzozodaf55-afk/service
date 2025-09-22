package handlers

import (
	"log"
	"strconv"
	"sync"
	"time"

	"action_users/constants"
	"action_users/controller"
	"action_users/models"
	"action_users/repositories"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	ctrl *controller.Controller
}

func NewHandler(ctrl *controller.Controller) *Handler {
	return &Handler{ctrl: ctrl}
}

func (h *Handler) ProcessUsers(c *fiber.Ctx) error {
	monthsStr := c.Query("months", "1")
	countryIdStr := c.Query("countryId", "0")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "50")

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

	from := (page - 1) * limit

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
				"months":                   months,
			},
		})
	}

	log.Printf("info: processing %d users (page %d, limit %d, months %d)",
		len(userIds), page, limit, months)

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
				clientHit, err := repositories.GetClientById(h.ctrl.Client(), uid, countryId)
				if err != nil {
					log.Printf("warn: getClientById(%s) error: %v", uid, err)
					return
				}
				if clientHit == nil {
					log.Printf("warn: registered user not found AND no actions for userId: %s", uid)
					return
				}
				cd := h.ctrl.BuildClientData(clientHit, nil, nil, nil, countryId, uid, actions, 0)
				mu.Lock()
				registeredNoActions = append(registeredNoActions, cd)
				mu.Unlock()
				return
			}

			isInactive := h.ctrl.CheckUserActionsInterval(actions, months)
			log.Printf("debug: user %s - actions: %d, isInactive: %t (months=%d)", uid, len(actions), isInactive, months)

			if isInactive {
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
					cd := h.ctrl.BuildClientData(nil, topUpSrc, betSrc, withdrawalSrc, countryId, uid, actions, months)
					mu.Lock()
					orphanUsers = append(orphanUsers, cd)
					mu.Unlock()
					return
				}

				cd := h.ctrl.BuildClientData(clientHit, topUpSrc, betSrc, withdrawalSrc, countryId, uid, actions, months)

				if cd.ReactivationThreshold > 0 {
					thresholdDate := time.Unix(cd.ReactivationThreshold, 0)
					lastActivityDate := time.Unix(cd.LastActivity, 0)
					log.Printf("info: user %s became inactive, last activity: %s, reactivation threshold: %s (months=%d)",
						uid, lastActivityDate.Format("2006-01-02"), thresholdDate.Format("2006-01-02"), months)
				}

				mu.Lock()
				inactiveResults = append(inactiveResults, cd)
				mu.Unlock()
			}
		}(uid)
	}
	wg.Wait()

	log.Printf("info: processed %d users: %d inactive, %d orphan, %d registered no actions (months=%d)",
		len(userIds), len(inactiveResults), len(orphanUsers), len(registeredNoActions), months)

	response := fiber.Map{
		"orphanUsers":         orphanUsers,
		"inactiveUsers":       inactiveResults,
		"registeredNoActions": registeredNoActions,
		"summary": fiber.Map{
			"orphanUsersCount":         len(orphanUsers),
			"inactiveUsersCount":       len(inactiveResults),
			"registeredNoActionsCount": len(registeredNoActions),
			"totalProcessed":           len(userIds),
			"page":                     page,
			"limit":                    limit,
			"months":                   months,
			"countryId":                countryId,
		},
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "Server is running",
	})
}
