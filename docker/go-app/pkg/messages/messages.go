package messages

type MsgID int

type Message struct {
	Id    MsgID  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"description"`
}
