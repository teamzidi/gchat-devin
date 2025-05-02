package chat

type Event struct {
	Type    string   `json:"type"`
	Message *Message `json:"message,omitempty"`
	User    *User    `json:"user,omitempty"`
	Space   *Space   `json:"space,omitempty"`
}

type Message struct {
	Name        string        `json:"name"`
	Sender      *User         `json:"sender"`
	Text        string        `json:"text"`
	Thread      *Thread       `json:"thread,omitempty"`
	Annotations []*Annotation `json:"annotations,omitempty"`
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email,omitempty"`
	Type        string `json:"type"`
}

type Space struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

type Thread struct {
	Name string `json:"name"`
}

type Annotation struct {
	Type        string `json:"type"`
	StartIndex  int    `json:"startIndex"`
	Length      int    `json:"length"`
	Text        string `json:"text,omitempty"`
	UserMention *User  `json:"userMention,omitempty"`
}

type Response struct {
	Text string `json:"text,omitempty"`
}

type ChatMessage struct {
	Text   string  `json:"text"`
	Thread *Thread `json:"thread,omitempty"`
}
