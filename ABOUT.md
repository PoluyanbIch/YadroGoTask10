# Система авторизации и рекомендаций для комиксов
## Описание
Этот проект представляет собой микросервисную систему, состоящую из двух основных компонентов:

1. AAA-сервис - управление аутентификацией, авторизацией и учётом пользовательской активности

2. Search-сервис - поиск комиксов и персонализированные рекомендации на основе анализа поведения пользователей

Система реализует полноценный AAA-фреймворк (Authentication, Authorization, Accounting) и интеллектуальную рекомендательную систему, использующую алгоритмы машинного обучения для анализа текстового контента.

## Архитектура 
### Сервис AAA
```go
type AAAService interface {
    // Аутентификация и управление токенами
    Login(ctx context.Context, login, password string) (accessToken, refreshToken string, expiresAt int64, user *User, err error)
    Register(ctx context.Context, login, password string) (user *User, err error)
    ValidateToken(ctx context.Context, token string) (userID int64, isAdmin bool, err error)
    RefreshToken(ctx context.Context, token string) (newAccessToken string, newRefreshToken string, expiresAt int64, err error)
    
    // Учёт пользовательской активности (Accounting)
    TrackSearch(ctx context.Context, userID int64, query string) error
    TrackComicView(ctx context.Context, userID int64, comicID int) error
    GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error)
    GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error)
}

type TokenManager interface {
    GenerateAccessToken(user *User) (string, int64, error) // token, expires_at
    GenerateRefreshToken(userID int64) (string, error)
    ParseAccessToken(token string) (*TokenClaims, error)
    ParseRefreshToken(token string) (userID int64, err error)
}
```

1. Двухтокенная аутентификация (Access + Refresh tokens)

2. JWT-токены с ролевой моделью (пользователь/администратор)

3. Трекинг пользовательской активности для сбора данных рекомендаций

### Сервис Search
```go
type SearchService interface {
    GetComic(context.Context, int) (Comic, error)
    GetRecommendations(context.Context, []string, []int, int) ([]Comic, error)
    GetLatestComics(context.Context, int) ([]Comic, error)
}
```

1. TF (Term Frequency) - как часто слово встречается в истории пользователя

2. IDF (Inverse Document Frequency) - насколько слово уникально во всей коллекции

3. Для каждого комикса строится аналогичный TF-IDF вектор, после чего вычисляется косинусное сходство:

```math
similarity(A,B) = cos(θ) = \frac{A·B}{||A|| × ||B||}
```
A - вектор профиля пользователя

B - вектор комикса

Результат от 0 (нет сходства) до 1 (полное сходство)

## Видео

![Видео](freestyle.mp4)