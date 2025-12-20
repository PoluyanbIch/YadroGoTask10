package core

type DBComic struct {
	ID          int
	URL         string
	Title       map[string]int
	Description map[string]int
	Alt         map[string]int
}

type Comic struct {
	ID  int
	URL string
}
