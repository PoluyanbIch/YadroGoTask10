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
