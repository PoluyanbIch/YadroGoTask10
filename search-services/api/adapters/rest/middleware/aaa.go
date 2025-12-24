package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"yadro.com/course/api/core"
)

func OptionalAuthMiddleware(next http.Handler, aaa core.AAA) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("access_token")
		if err != nil {
			// Нет куки - просто продолжаем без авторизации
			next.ServeHTTP(w, r)
			return
		}

		// Если токен есть - валидируем и добавляем в контекст
		if token.Value != "" {
			userID, isAdmin, err := aaa.ValidateToken(r.Context(), token.Value)
			if err == nil {
				// Токен валиден - добавляем в контекст
				ctx := r.Context()
				ctx = context.WithValue(ctx, "userID", userID)
				ctx = context.WithValue(ctx, "isAdmin", isAdmin)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			refreshToken, refreshErr := r.Cookie("refresh_token")
			if refreshErr == nil && refreshToken.Value != "" {
				newAccess, newRefresh, expiresAt, refreshErr := aaa.RefreshToken(r.Context(), refreshToken.Value)
				if refreshErr == nil {
					// Устанавливаем новые куки
					setAuthCookies(w, newAccess, newRefresh, aaa, expiresAt)

					// Валидируем новый access токен
					userID, isAdmin, err := aaa.ValidateToken(r.Context(), newAccess)
					if err == nil {
						ctx := r.Context()
						ctx = context.WithValue(ctx, "userID", userID)
						ctx = context.WithValue(ctx, "isAdmin", isAdmin)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
			// Токен невалиден - просто продолжаем без userID
		}

		// Нет токена или токен невалиден - продолжаем без авторизации
		next.ServeHTTP(w, r)
	})
}

func RequiredAuthMiddleware(next http.Handler, aaa core.AAA) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("access_token")
		if err != nil || token == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if token.Value == "" {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		userID, isAdmin, err := aaa.ValidateToken(r.Context(), token.Value)
		if err != nil {
			// Если токен истек - пытаемся обновить
			refreshToken, refreshErr := r.Cookie("refresh_token")
			if refreshErr == nil && refreshToken.Value != "" {
				newAccess, newRefresh, expiresAt, refreshErr := aaa.RefreshToken(r.Context(), refreshToken.Value)
				if refreshErr == nil {
					// Устанавливаем новые куки
					setAuthCookies(w, newAccess, newRefresh, aaa, expiresAt)

					// Валидируем новый access токен
					userID, isAdmin, err := aaa.ValidateToken(r.Context(), newAccess)
					if err == nil {
						ctx := r.Context()
						ctx = context.WithValue(ctx, "userID", userID)
						ctx = context.WithValue(ctx, "isAdmin", isAdmin)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
			http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", userID)
		ctx = context.WithValue(ctx, "isAdmin", isAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminMiddleware(next http.Handler, aaa core.AAA) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if token == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, isAdmin, err := aaa.ValidateToken(r.Context(), token.Value)
		if err != nil {
			// Если токен истек - пытаемся обновить
			refreshToken, refreshErr := r.Cookie("refresh_token")
			if refreshErr == nil && refreshToken.Value != "" {
				newAccess, newRefresh, expiresAt, refreshErr := aaa.RefreshToken(r.Context(), refreshToken.Value)
				if refreshErr == nil {
					// Устанавливаем новые куки
					setAuthCookies(w, newAccess, newRefresh, aaa, expiresAt)

					// Валидируем новый access токен
					userID, isAdmin, err = aaa.ValidateToken(r.Context(), newAccess)
					if err != nil {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
				} else {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		ctx = context.WithValue(ctx, "isAdmin", isAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, aaa core.AAA, expiresAt int64) {
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Только HTTPS в production
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(expiresAt, 0),
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/auth/refresh",
		HttpOnly: true,
		Secure:   true, // Только HTTPS в production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 дней
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

// TrackedResponseWriter перехватывает статус ответа
type TrackedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *TrackedResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

// WithTracking возвращает middleware для трекинга успешных запросов
func WithTracking(next http.HandlerFunc, aaa core.AAA, log *slog.Logger, trackType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Создаем tracked writer для перехвата статуса
		tracked := &TrackedResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Выполняем основной handler
		next(tracked, r)

		// Трекаем только если запрос успешен (2xx)
		if tracked.statusCode >= 200 && tracked.statusCode < 300 {
			trackRequest(r, aaa, log, trackType)
		}
	}
}

func trackRequest(r *http.Request, aaa core.AAA, log *slog.Logger, trackType string) {
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		return // Не авторизован - не трекаем
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		return
	}

	// Создаем новый контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch trackType {
	case "search":
		query := r.URL.Query().Get("phrase")
		if query == "" {
			return
		}

		if err := aaa.TrackSearch(ctx, userID, query); err != nil {
			log.Error("failed to track search",
				"user_id", userID,
				"query", query,
				"error", err)
		} else {
			log.Debug("search tracked",
				"user_id", userID,
				"query", query)
		}

	case "comic_view":
		comicIDStr := r.PathValue("id")
		if comicIDStr == "" {
			comicIDStr = r.URL.Query().Get("id")
		}

		if comicIDStr == "" {
			return
		}

		comicID, err := strconv.Atoi(comicIDStr)
		if err != nil {
			return
		}

		if err := aaa.TrackComicView(ctx, userID, comicID); err != nil {
			log.Error("failed to track comic view",
				"user_id", userID,
				"comic_id", comicID,
				"error", err)
		} else {
			log.Debug("comic view tracked",
				"user_id", userID,
				"comic_id", comicID)
		}
	}
}

// WithSearchTracking упрощенный вариант для поиска
func WithSearchTracking(next http.HandlerFunc, aaa core.AAA, log *slog.Logger) http.HandlerFunc {
	return WithTracking(next, aaa, log, "search")
}

// WithComicViewTracking для просмотра комиксов
func WithComicViewTracking(next http.HandlerFunc, aaa core.AAA, log *slog.Logger) http.HandlerFunc {
	return WithTracking(next, aaa, log, "comic_view")
}
