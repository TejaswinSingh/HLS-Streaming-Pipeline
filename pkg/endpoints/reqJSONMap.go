package endpoints

type ServiceStatusRequest struct{}
type ServiceStatusResponse struct {
	Status int `json:"status"`
}

func (r ServiceStatusResponse) GetStatus() int { return r.Status }
