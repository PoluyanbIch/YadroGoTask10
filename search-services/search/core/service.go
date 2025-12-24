package core

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"strings"
)

type Service struct {
	log       *slog.Logger
	db        DB
	words     Words
	index     map[string][]int
	comicsMap map[int]DBComic
}

func NewService(log *slog.Logger, db DB, words Words) *Service {
	s := &Service{
		log:       log,
		db:        db,
		words:     words,
		index:     nil,
		comicsMap: nil,
	}
	return s
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comic, error) {
	normPhrase, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("Search/words error", "error", err)
		return nil, err
	}
	comics, err := s.db.Read(ctx)
	if err != nil {
		s.log.Error("Search/db error", "error", err)
		return nil, err
	}
	comicsMap := make(map[int]DBComic)
	for _, comic := range comics {
		comicsMap[comic.ID] = comic
	}
	res := search(normPhrase, comics)

	type result struct {
		ID    int
		Score float64
	}
	var sortedResult []result
	for id, score := range res {
		sortedResult = append(sortedResult, result{ID: id, Score: score})
	}
	sort.Slice(sortedResult, func(i, j int) bool { return sortedResult[i].Score > sortedResult[j].Score })
	var searchResult []Comic
	for _, i := range sortedResult {
		searchResult = append(searchResult, Comic{ID: i.ID, URL: comicsMap[i.ID].URL})
	}
	if len(searchResult) < limit {
		limit = len(searchResult)
	}
	return searchResult[:limit], nil
}

func (s *Service) ISearch(ctx context.Context, phrase string, limit int) ([]Comic, error) {
	normPhrase, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("norm phrase error", "error", err)
		return nil, err
	}

	idfCache := make(map[string]float64)
	for _, word := range normPhrase {
		idfCache[word] = indexCalculateIDF(word, s.index[word], s.comicsMap)
	}

	const (
		titleWeight       = 3.0
		altWeight         = 2.0
		descriptionWeight = 1.0
		fullMatchBonus    = 100.0
	)
	res := make(map[int]float64)
	comicsWithWords := make(map[int]struct{})
	for _, word := range normPhrase {
		for _, id := range s.index[word] {
			comicsWithWords[id] = struct{}{}
		}
	}
	for id := range comicsWithWords {
		res[id] = 0
		countMatchWords := 0
		for _, word := range normPhrase {
			res[id] += idfCache[word] * calculateTF(word, s.comicsMap[id].Title) * titleWeight
			res[id] += idfCache[word] * calculateTF(word, s.comicsMap[id].Alt) * altWeight
			res[id] += idfCache[word] * calculateTF(word, s.comicsMap[id].Description) * descriptionWeight
			switch {
			case countSubstringInField(word, s.comicsMap[id].Title) > 0:
				countMatchWords++
			case countSubstringInField(word, s.comicsMap[id].Alt) > 0:
				countMatchWords++
			case countSubstringInField(word, s.comicsMap[id].Description) > 0:
				countMatchWords++
			}
		}
		if countMatchWords == len(normPhrase) {
			res[id] *= fullMatchBonus
		}
	}
	type result struct {
		ID    int
		Score float64
	}
	var sortedResult []result
	for id, score := range res {
		sortedResult = append(sortedResult, result{ID: id, Score: score})
	}
	sort.Slice(sortedResult, func(i, j int) bool { return sortedResult[i].Score > sortedResult[j].Score })
	var searchResult []Comic
	for _, i := range sortedResult {
		searchResult = append(searchResult, Comic{ID: i.ID, URL: s.comicsMap[i.ID].URL})
	}
	if len(searchResult) < limit {
		limit = len(searchResult)
	}
	return searchResult[:limit], nil
}

func (s *Service) BuildIndex(ctx context.Context) error {
	comics, err := s.db.Read(ctx)
	if err != nil {
		s.log.Error("DB Read error", "error", err)
		return err
	}
	comicsMap := make(map[int]DBComic)
	index := make(map[string][]int)
	for _, c := range comics {
		comicsMap[c.ID] = c
		words := mergeMaps(c.Alt, mergeMaps(c.Description, c.Title))
		for word := range words {
			index[word] = append(index[word], c.ID)
		}
	}
	s.index = index
	s.comicsMap = comicsMap
	return nil
}

func mergeMaps(map1 map[string]int, map2 map[string]int) map[string]int {
	result := make(map[string]int)
	for key, el := range map2 {
		result[key] += el
	}
	for key, el := range map1 {
		result[key] += el
	}
	return result
}

func search(phrase []string, comics []DBComic) map[int]float64 {
	res := make(map[int]float64)

	comicsMap := make(map[int]DBComic)
	for _, comic := range comics {
		comicsMap[comic.ID] = comic
	}

	var comicsWithWords []int
	for _, c := range comics {
		for _, word := range phrase {
			if countSubstringInField(word, c.Description) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
			if countSubstringInField(word, c.Alt) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
			if countSubstringInField(word, c.Title) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
		}
	}

	idfCache := make(map[string]float64)
	for _, word := range phrase {
		idfCache[word] = calculateIDF(word, comics)
	}
	const (
		titleWeight       = 3.0
		altWeight         = 2.0
		descriptionWeight = 1.0
		fullMatchBonus    = 100.0
	)

	for _, id := range comicsWithWords {

		res[id] = 0
		countMatchWords := 0
		for _, word := range phrase {
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Title) * titleWeight
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Alt) * altWeight
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Description) * descriptionWeight
			switch {
			case countSubstringInField(word, comicsMap[id].Title) > 0:
				countMatchWords++
			case countSubstringInField(word, comicsMap[id].Alt) > 0:
				countMatchWords++
			case countSubstringInField(word, comicsMap[id].Description) > 0:
				countMatchWords++
			}
		}
		if countMatchWords == len(phrase) {
			res[id] *= fullMatchBonus
		}
	}
	return res
}

func calculateTF(word string, field map[string]int) float64 {
	if field == nil {
		return 0
	}
	num := countSubstringInField(word, field)
	sum := 0
	for _, val := range field {
		sum += val
	}
	if sum == 0 {
		return 0
	}
	return float64(num) / float64(sum)
}

func indexCalculateIDF(word string, comicIds []int, comicsMap map[int]DBComic) float64 {
	num := 0
	for _, comic := range comicsMap {
		title := comic.Title
		alt := comic.Alt
		description := comic.Description
		switch {
		case countSubstringInField(word, title) > 0:
			num++
		case countSubstringInField(word, alt) > 0:
			num++
		case countSubstringInField(word, description) > 0:
			num++
		}
	}
	if num == 0 {
		return 0
	}
	sum := len(comicsMap)
	return math.Log(float64(sum) / float64(num))
}

func calculateIDF(word string, comics []DBComic) float64 {
	num := 0
	for _, comic := range comics {
		title := comic.Title
		alt := comic.Alt
		description := comic.Description
		switch {
		case countSubstringInField(word, title) > 0:
			num++
		case countSubstringInField(word, alt) > 0:
			num++
		case countSubstringInField(word, description) > 0:
			num++
		}
	}
	if num == 0 {
		return 0
	}
	sum := len(comics)
	return math.Log(float64(sum) / float64(num))
}

func countSubstringInField(word string, field map[string]int) int {
	num := 0
	for w := range field {
		if strings.Contains(word, w) || strings.Contains(w, word) {
			num += field[w]
		}
	}
	return num
}

func (s *Service) GetComic(ctx context.Context, comicID int) (Comic, error) {
	dbComic, ok := s.comicsMap[comicID]
	if !ok {
		return Comic{}, nil
	}
	return Comic{ID: dbComic.ID, URL: dbComic.URL}, nil
}

func (s *Service) GetRecommendations(ctx context.Context, searchHistory []string, viewedComics []int, limit int) ([]Comic, error) {
	if len(viewedComics) == 0 && len(searchHistory) == 0 {
		// Нет данных - возвращаем популярные
		return nil, nil
	}

	// 1. Строим профиль пользователя на основе просмотренных комиксов
	userProfile := s.buildUserProfile(viewedComics)

	// 2. Добавляем слова из истории поиска в профиль
	for _, query := range searchHistory {
		normWords, err := s.words.Norm(ctx, query)
		if err == nil {
			for _, word := range normWords {
				userProfile[word] += 3.0 // Даем больший вес поисковым запросам
			}
		}
	}

	// 3. Ищем комиксы, похожие на профиль пользователя
	recommendations := s.findComicsByProfile(userProfile, viewedComics, limit)
	result := make([]Comic, 0, len(recommendations))
	for _, comicID := range recommendations {
		dbComic, ok := s.comicsMap[comicID]
		if ok {
			result = append(result, Comic{ID: dbComic.ID, URL: dbComic.URL})
		}
	}
	return result, nil
}

// Строит профиль пользователя как вектор TF-IDF слов
func (s *Service) buildUserProfile(viewedComics []int) map[string]float64 {
	profile := make(map[string]float64)

	if len(viewedComics) == 0 {
		return profile
	}

	// Собираем все слова из просмотренных комиксов
	wordCounts := make(map[string]int)
	totalWords := 0

	for _, comicID := range viewedComics {
		comic, ok := s.comicsMap[comicID]
		if !ok {
			continue
		}

		// Считаем слова из title, alt, description
		for word := range comic.Title {
			wordCounts[word]++
			totalWords++
		}
		for word := range comic.Alt {
			wordCounts[word]++
			totalWords++
		}
		for word := range comic.Description {
			wordCounts[word]++
			totalWords++
		}
	}

	// Преобразуем в TF (Term Frequency)
	for word, count := range wordCounts {
		tf := float64(count) / float64(totalWords)

		// Умножаем на IDF слова
		idf := s.calculateIDFRecommendations(word)
		profile[word] = tf * idf
	}

	return profile
}

// Ищет комиксы, похожие на профиль пользователя
func (s *Service) findComicsByProfile(userProfile map[string]float64, viewedComics []int, limit int) []int {
	type score struct {
		id    int
		score float64
	}

	var scores []score
	viewedSet := make(map[int]bool)
	for _, id := range viewedComics {
		viewedSet[id] = true
	}

	// Для каждого комикса вычисляем косинусное сходство с профилем пользователя
	for comicID, comic := range s.comicsMap {
		if viewedSet[comicID] {
			continue // Пропускаем уже просмотренные
		}

		comicVector := s.buildComicVector(comic)
		similarity := cosineSimilarity(userProfile, comicVector)

		if similarity > 0 {
			scores = append(scores, score{id: comicID, score: similarity})
		}
	}

	// Сортируем по убыванию сходства
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Берем top N
	result := make([]int, 0, limit)
	for i := 0; i < len(scores) && i < limit; i++ {
		result = append(result, scores[i].id)
	}

	return result
}

// Строит вектор TF-IDF для комикса
func (s *Service) buildComicVector(comic DBComic) map[string]float64 {
	vector := make(map[string]float64)

	// Собираем все слова комикса
	wordCounts := make(map[string]int)
	totalWords := 0

	for word, count := range comic.Title {
		wordCounts[word] += count * 3 // Больший вес для title
		totalWords += count * 3
	}
	for word, count := range comic.Alt {
		wordCounts[word] += count * 2 // Средний вес для alt
		totalWords += count * 2
	}
	for word, count := range comic.Description {
		wordCounts[word] += count
		totalWords += count
	}

	// Преобразуем в TF-IDF
	for word, count := range wordCounts {
		tf := float64(count) / float64(totalWords)
		idf := s.calculateIDFRecommendations(word)
		vector[word] = tf * idf
	}

	return vector
}

// Косинусное сходство между двумя векторами
func cosineSimilarity(vec1, vec2 map[string]float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0
	}

	dotProduct := 0.0
	magnitude1 := 0.0
	magnitude2 := 0.0

	// Вычисляем по словам, которые есть в обоих векторах
	for word, val1 := range vec1 {
		magnitude1 += val1 * val1
		if val2, ok := vec2[word]; ok {
			dotProduct += val1 * val2
		}
	}

	for _, val2 := range vec2 {
		magnitude2 += val2 * val2
	}

	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)

	if magnitude1 == 0 || magnitude2 == 0 {
		return 0
	}

	return dotProduct / (magnitude1 * magnitude2)
}
func (s *Service) calculateIDFRecommendations(word string) float64 {
	// Используем уже построенный индекс
	comicIDs, exists := s.index[word]
	if !exists || len(comicIDs) == 0 {
		return 0
	}

	totalComics := len(s.comicsMap)
	comicsWithWord := len(comicIDs)

	return math.Log(float64(totalComics) / float64(comicsWithWord))
}

func (s *Service) GetLatestComics(ctx context.Context, limit int) ([]Comic, error) {
	// Получаем все ID комиксов
	var allIDs []int
	for id := range s.comicsMap {
		allIDs = append(allIDs, id)
	}

	// Сортируем по ID (предполагая что больший ID = новее)
	sort.Slice(allIDs, func(i, j int) bool {
		return allIDs[i] > allIDs[j] // DESC order
	})

	// Берем top N
	if len(allIDs) < limit {
		limit = len(allIDs)
	}
	var latestComics []Comic
	for i := 0; i < limit; i++ {
		id := allIDs[i]
		dbComic := s.comicsMap[id]
		latestComics = append(latestComics, Comic{ID: dbComic.ID, URL: dbComic.URL})
	}
	return latestComics, nil
}
