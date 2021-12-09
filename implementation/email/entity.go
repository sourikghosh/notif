package email

type NameAddr struct {
	EmailAddr string
	UserName  string
}

type Entity struct {
	FromName string
	ToList   []NameAddr
	Subject  string
	Body     string
}
