package licensing

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/http/render"
)

// noopAPI is a no-op implementation of the API interface for the base server.
// The enterprise server overrides this with the httplicensing API.
type noopAPI struct{}

func NewNoopAPI() API {
	return &noopAPI{}
}

func (api *noopAPI) Activate(rw http.ResponseWriter, r *http.Request) {
	render.Success(rw, http.StatusOK, nil)
}

func (api *noopAPI) Refresh(rw http.ResponseWriter, r *http.Request) {
	render.Success(rw, http.StatusOK, nil)
}

func (api *noopAPI) GetActive(rw http.ResponseWriter, r *http.Request) {
	render.Success(rw, http.StatusOK, nil)
}

func (api *noopAPI) Checkout(rw http.ResponseWriter, r *http.Request) {
	render.Success(rw, http.StatusOK, nil)
}

func (api *noopAPI) Portal(rw http.ResponseWriter, r *http.Request) {
	render.Success(rw, http.StatusOK, nil)
}
