package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		replies := make(map[string]string)

		for name, pinger := range pingers {
			err := pinger.Ping(ctx)
			if err != nil {
				log.Warn("ping failed", "service", name, "error", err)
				replies[name] = "unavailable"
			} else {
				replies[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		resp := PingResponse{Replies: replies}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write ping response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewWordsHandler(log *slog.Logger, norm core.Normalizer, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		wordsList, err := norm.Norm(ctx, phrase)
		if err != nil {
			handleError(w, err)
			return
		}

		resp := map[string]interface{}{
			"words": wordsList,
			"total": len(wordsList),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write response", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrUpdateInProgress):
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, core.ErrBadArguments):
		http.Error(w, "phrase too large", http.StatusBadRequest)
	case errors.Is(err, core.ErrServiceUnavailable):
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	default:
		http.Error(w, "internal service error", http.StatusInternalServerError)
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		if err := updater.Update(ctx); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type ServiceStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		stats, err := updater.Stats(ctx)
		if err != nil {
			handleError(w, err)
			return
		}

		statsJson := ServiceStats{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(statsJson); err != nil {
			log.Error("failed to write response", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		status, err := updater.Status(ctx)
		if err != nil {
			handleError(w, err)
			return
		}

		reply := map[string]string{
			"status": string(status),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Error("cannot encode status reply")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		if err := updater.Drop(ctx); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, `{"error": "missing phrase"}`, http.StatusBadRequest)
			return
		}

		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, `{"error": "invalid limit format"}`, http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				http.Error(w, `{"error": "limit must be positive"}`, http.StatusBadRequest)
				return
			}
			limit = parsed
		}

		searchResp, err := searcher.Search(ctx, phrase, limit)
		if err != nil {
			handleError(w, err)
			return
		}

		type Comic struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		}

		type SearchResponse struct {
			Comics []Comic `json:"comics"`
			Total  int     `json:"total"`
		}

		comics := make([]Comic, len(searchResp.Comics))
		for i, comic := range searchResp.Comics {
			comics[i] = Comic{
				ID:  comic.Id,
				URL: comic.Url,
			}
		}

		resp := SearchResponse{
			Comics: comics,
			Total:  len(searchResp.Comics),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(resp); err != nil {
			log.Error("failed to encode search response", "error", err)
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		}
	}
}

func NewISearchHandler(log *slog.Logger, searcher core.Searcher, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, `{"error": "missing phrase"}`, http.StatusBadRequest)
			return
		}

		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, `{"error": "invalid limit format"}`, http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				http.Error(w, `{"error": "limit must be positive"}`, http.StatusBadRequest)
				return
			}
			limit = parsed
		}

		searchResp, err := searcher.ISearch(ctx, phrase, limit)
		if err != nil {
			handleError(w, err)
			return
		}

		type Comic struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		}

		type SearchResponse struct {
			Comics []Comic `json:"comics"`
			Total  int     `json:"total"`
		}

		comics := make([]Comic, len(searchResp.Comics))
		for i, comic := range searchResp.Comics {
			comics[i] = Comic{
				ID:  comic.Id,
				URL: comic.Url,
			}
		}

		resp := SearchResponse{
			Comics: comics,
			Total:  len(searchResp.Comics),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(resp); err != nil {
			log.Error("failed to encode search response", "error", err)
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		}
	}
}

func NewLoginHandler(log *slog.Logger, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		creds.Login = r.FormValue("login")
		creds.Password = r.FormValue("password")
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()
		accessToken, refreshToken, expiresAt, user, err := aaa.Login(ctx, creds.Login, creds.Password)
		if err != nil {
			handleError(w, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  time.Unix(expiresAt, 0),
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/auth/refresh",
			MaxAge:   30 * 24 * 60 * 60,
		})

		resp := map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_at":    expiresAt,
			"user": map[string]interface{}{
				"id":       user.ID,
				"login":    user.Login,
				"is_admin": user.IsAdmin,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write login response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewRegisterHandler(log *slog.Logger, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		creds.Login = r.FormValue("login")
		creds.Password = r.FormValue("password")
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()
		user, err := aaa.Register(ctx, creds.Login, creds.Password)
		if err != nil {
			handleError(w, err)
			return
		}
		resp := map[string]interface{}{
			"user": map[string]interface{}{
				"id":       user.ID,
				"login":    user.Login,
				"is_admin": user.IsAdmin,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write register response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewRefreshHandler(log *slog.Logger, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()
		accessToken, refreshToken, expiresAt, err := aaa.RefreshToken(ctx, req.RefreshToken)
		if err != nil {
			handleError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  time.Unix(expiresAt, 0),
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/auth/refresh",
			MaxAge:   30 * 24 * 60 * 60,
		})

		resp := map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_at":    expiresAt,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write refresh response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewLogoutHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    "",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusOK)
	}
}

func NewGetRecentSearchesHandler(log *slog.Logger, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value("userID")
		if userIDVal == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		userID, ok := userIDVal.(int64)
		if !ok {
			http.Error(w, `{"error": "invalid user id"}`, http.StatusUnauthorized)
			return
		}
		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, `{"error": "invalid limit format"}`, http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				http.Error(w, `{"error": "limit must be positive"}`, http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		searches, err := aaa.GetRecentSearches(r.Context(), userID, limit)
		if err != nil {
			handleError(w, err)
			return
		}
		resp := map[string]interface{}{
			"queries": searches,
			"total":   len(searches),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write recent searches response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewGetRecentViewsHandler(log *slog.Logger, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value("userID")
		if userIDVal == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		userID, ok := userIDVal.(int64)
		if !ok {
			http.Error(w, `{"error": "invalid user id"}`, http.StatusUnauthorized)
			return
		}
		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, `{"error": "invalid limit format"}`, http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				http.Error(w, `{"error": "limit must be positive"}`, http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		views, err := aaa.GetRecentViews(r.Context(), userID, limit)
		if err != nil {
			handleError(w, err)
			return
		}
		resp := map[string]interface{}{
			"comic_ids": views,
			"total":     len(views),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write recent views response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewGetComicHandler(log *slog.Logger, searcher core.Searcher, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comicIDStr := r.URL.Query().Get("id")
		if comicIDStr == "" {
			http.Error(w, `{"error": "missing comic id"}`, http.StatusBadRequest)
			return
		}
		comicID, err := strconv.Atoi(comicIDStr)
		if err != nil {
			http.Error(w, `{"error": "invalid comic id format"}`, http.StatusBadRequest)
			return
		}
		comic, err := searcher.GetComic(r.Context(), comicID)
		if err != nil {
			handleError(w, err)
			return
		}
		resp := map[string]interface{}{
			"id":  comic.Id,
			"url": comic.Url,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write get comic response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewGetRecommendationsHandler(log *slog.Logger, searcher core.Searcher, aaa core.AAA, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDVal := r.Context().Value("userID")
		if userIDVal == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		userID, ok := userIDVal.(int64)
		if !ok {
			http.Error(w, `{"error": "invalid user id"}`, http.StatusUnauthorized)
			return
		}
		var req struct {
			Limit int `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				req.Limit = 10
			} else {
				http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
				return
			}
		}
		if req.Limit <= 0 {
			req.Limit = 10
		}
		searchHistory, err := aaa.GetRecentSearches(r.Context(), userID, req.Limit)
		if err != nil {
			handleError(w, err)
			return
		}
		viewHistory, err := aaa.GetRecentViews(r.Context(), userID, req.Limit)
		if err != nil {
			handleError(w, err)
			return
		}
		recommendations, err := searcher.GetRecommendations(r.Context(), searchHistory, viewHistory, req.Limit)
		if err != nil {
			handleError(w, err)
			return
		}
		comics := []map[string]interface{}{}
		for _, c := range recommendations.Comics {
			comics = append(comics, map[string]interface{}{
				"id":  c.Id,
				"url": c.Url,
			})
		}
		resp := map[string]interface{}{
			"comics": comics,
			"total":  recommendations.Total,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write get recommendations response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewGetLatestComicsHandler(log *slog.Logger, searcher core.Searcher, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 10
		limitStr := r.PathValue("limit")
		if limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, `{"error": "invalid limit format"}`, http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				http.Error(w, `{"error": "limit must be positive"}`, http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		searchResp, err := searcher.GetLatestComics(r.Context(), limit)
		if err != nil {
			handleError(w, err)
			return
		}
		type Comic struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		}

		type SearchResponse struct {
			Comics []Comic `json:"comics"`
			Total  int     `json:"total"`
		}
		comics := make([]Comic, len(searchResp.Comics))
		for i, comic := range searchResp.Comics {
			comics[i] = Comic{
				ID:  comic.Id,
				URL: comic.Url,
			}
		}
		resp := SearchResponse{
			Comics: comics,
			Total:  len(searchResp.Comics),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(resp); err != nil {
			log.Error("failed to encode latest comics response", "error", err)
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		}
	}
}
