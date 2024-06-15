package dto

type RoomDTO struct {
	ID       string      `json:"id"`
	VideoUrl *string     `json:"videoUrl"`
	Members  []MemberDTO `json:"members"`
}
