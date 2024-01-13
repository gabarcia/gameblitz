package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabarcia/metagaming-api/internal/leaderboard"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildGetLeaderboardMiddleware(t *testing.T) {
	var (
		leaderboardID = uuid.NewString()
		gameID        = uuid.NewString()
	)

	t.Run("OK", func(t *testing.T) {
		getLeaderboardMiddleware := BuildGetLeaderboardMiddleware(nil, time.Minute, func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
			return leaderboard.Leaderboard{ID: id, GameID: gameID}, nil
		})

		app := fiber.New()
		app.Get("/:leaderboardId", getLeaderboardMiddleware, func(c *fiber.Ctx) error {
			leaderboard := c.Locals("leaderboard")
			return c.Status(http.StatusOK).JSON(leaderboard)
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", leaderboardID), nil)

		req.Header.Set(gameIDHeader, gameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body Leaderboard
		err = json.NewDecoder(resp.Body).Decode(&body)
		assert.NoError(t, err)

		assert.Equal(t, leaderboardID, body.ID)
		assert.Equal(t, gameID, body.GameID)
	})

	t.Run("Missing Game ID", func(t *testing.T) {
		getLeaderboardMiddleware := BuildGetLeaderboardMiddleware(nil, time.Minute, func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
			return leaderboard.Leaderboard{ID: id, GameID: gameID}, nil
		})

		app := fiber.New()
		app.Get("/:leaderboardId", getLeaderboardMiddleware, func(c *fiber.Ctx) error {
			leaderboard := c.Locals("leaderboard")
			return c.Status(http.StatusOK).JSON(leaderboard)
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", leaderboardID), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var body ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Code, body.Code)
		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Message, body.Message)
	})

	t.Run("Leaderboard Not Found", func(t *testing.T) {
		getLeaderboardMiddleware := BuildGetLeaderboardMiddleware(nil, time.Minute, func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
			return leaderboard.Leaderboard{}, leaderboard.ErrLeaderboardNotFound
		})

		app := fiber.New(fiber.Config{ErrorHandler: BuildErrorHandler()})
		app.Get("/:leaderboardId", getLeaderboardMiddleware, func(c *fiber.Ctx) error {
			leaderboard := c.Locals("leaderboard")
			return c.Status(http.StatusOK).JSON(leaderboard)
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", leaderboardID), nil)

		req.Header.Set(gameIDHeader, gameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var body ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardNotFound.Code, body.Code)
		assert.Equal(t, ErrorResponseLeaderboardNotFound.Message, body.Message)
	})
}

func TestBuildCreateLeaderboardHandler(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		app := App(Config{
			CreateLeaderboardFunc: leaderboard.BuildCreateFunc(func(ctx context.Context, data leaderboard.NewLeaderboardData) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{ID: uuid.NewString()}, nil
			}),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/leaderboards", bytes.NewBufferString(`{
			"gameId": "66868dc7-d391-418d-b9f1-a85a4fd096e4",
			"name": "Test Leaderboard",
			"description": "Test create leaderboard request",
			"startAt": "2024-01-01T00:00:00Z",
			"endAt": null,
			"aggregationMode": "MAX",
			"dataType": "INT",
			"ordering": "DESC"
		}`))

		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var data Leaderboard
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.NotEmpty(t, data.ID)
	})

	t.Run("Validation Error", func(t *testing.T) {
		app := App(Config{
			CreateLeaderboardFunc: leaderboard.BuildCreateFunc(func(ctx context.Context, data leaderboard.NewLeaderboardData) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{ID: uuid.NewString()}, nil
			}),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/leaderboards", bytes.NewBufferString(`{}`))

		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardInvalid.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardInvalid.Message, data.Message)
		assert.NotEmpty(t, data.Details)
	})

	t.Run("Random Error", func(t *testing.T) {
		app := App(Config{
			CreateLeaderboardFunc: leaderboard.BuildCreateFunc(func(ctx context.Context, data leaderboard.NewLeaderboardData) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{}, errors.New("any error")
			}),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/leaderboards", bytes.NewBufferString(`{
			"gameId": "66868dc7-d391-418d-b9f1-a85a4fd096e4",
			"name": "Test Leaderboard",
			"description": "Test create leaderboard request",
			"startAt": "2024-01-01T00:00:00Z",
			"endAt": null,
			"aggregationMode": "MAX",
			"dataType": "INT",
			"ordering": "DESC"
		}`))

		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseInternalServerError.Code, data.Code)
		assert.Equal(t, ErrorResponseInternalServerError.Message, data.Message)
		assert.Empty(t, data.Details)
	})
}

func TestBuildGetLeaderboardHandler(t *testing.T) {
	var (
		expectedID     = uuid.NewString()
		expectedGameID = uuid.NewString()
	)

	t.Run("OK", func(t *testing.T) {
		app := App(Config{
			GetLeaderboardByIDAndGameIDFunc: leaderboard.BuildGetByIDAndGameIDFunc(func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{ID: id, GameID: gameID}, nil
			}),
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var data Leaderboard
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, expectedID, data.ID)
		assert.Equal(t, expectedGameID, data.GameID)
	})

	t.Run("Missing Game ID", func(t *testing.T) {
		app := App(Config{
			GetLeaderboardByIDAndGameIDFunc: leaderboard.BuildGetByIDAndGameIDFunc(func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{}, nil
			}),
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Message, data.Message)
	})

	t.Run("Not Found", func(t *testing.T) {
		app := App(Config{
			GetLeaderboardByIDAndGameIDFunc: leaderboard.BuildGetByIDAndGameIDFunc(func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{}, leaderboard.ErrLeaderboardNotFound
			}),
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardNotFound.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardNotFound.Message, data.Message)
	})

	t.Run("Invalid Leaderboard ID", func(t *testing.T) {
		app := App(Config{
			GetLeaderboardByIDAndGameIDFunc: leaderboard.BuildGetByIDAndGameIDFunc(func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{}, leaderboard.ErrInvalidLeaderboardID
			}),
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardInvalidID.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardInvalidID.Message, data.Message)
	})

	t.Run("Random Error", func(t *testing.T) {
		app := App(Config{
			GetLeaderboardByIDAndGameIDFunc: leaderboard.BuildGetByIDAndGameIDFunc(func(ctx context.Context, id, gameID string) (leaderboard.Leaderboard, error) {
				return leaderboard.Leaderboard{}, errors.New("any error")
			}),
		})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseInternalServerError.Code, data.Code)
		assert.Equal(t, ErrorResponseInternalServerError.Message, data.Message)
	})
}

func TestBuildDeleteLeaderboardHandler(t *testing.T) {
	var (
		expectedID     = uuid.NewString()
		expectedGameID = uuid.NewString()
	)

	t.Run("OK", func(t *testing.T) {
		app := App(Config{
			DeleteLeaderboardByIDAndGameIDFunc: leaderboard.BuildSoftDeleteFunc(func(ctx context.Context, id, gameID string) error {
				return nil
			}),
		})

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Missing Game ID", func(t *testing.T) {
		app := App(Config{
			DeleteLeaderboardByIDAndGameIDFunc: leaderboard.BuildSoftDeleteFunc(func(ctx context.Context, id, gameID string) error {
				return nil
			}),
		})

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardInvalidGameID.Message, data.Message)
	})

	t.Run("Not Found", func(t *testing.T) {
		app := App(Config{
			DeleteLeaderboardByIDAndGameIDFunc: leaderboard.BuildSoftDeleteFunc(func(ctx context.Context, id, gameID string) error {
				return leaderboard.ErrLeaderboardNotFound
			}),
		})

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseLeaderboardNotFound.Code, data.Code)
		assert.Equal(t, ErrorResponseLeaderboardNotFound.Message, data.Message)
	})

	t.Run("Random Error", func(t *testing.T) {
		app := App(Config{
			DeleteLeaderboardByIDAndGameIDFunc: leaderboard.BuildSoftDeleteFunc(func(ctx context.Context, id, gameID string) error {
				return errors.New("any error")
			}),
		})

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/leaderboards/%s", expectedID), nil)

		req.Header.Set(gameIDHeader, expectedGameID)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var data ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		assert.NoError(t, err)

		assert.Equal(t, ErrorResponseInternalServerError.Code, data.Code)
		assert.Equal(t, ErrorResponseInternalServerError.Message, data.Message)
	})
}
