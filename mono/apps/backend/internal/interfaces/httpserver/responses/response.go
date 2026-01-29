package responses

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NewInternalServerError sends a 500 error response
func NewInternalServerError(reqCtx *gin.Context, errResp ErrorResponse) {
	if errResp.ErrorInstance != nil {
		reqCtx.Error(errResp.ErrorInstance)
	}
	reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, errResp)
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}

type PageCursor struct {
	FirstID *string
	LastID  *string
	HasMore bool
	Total   int64
}

func BuildCursorPage[T any](
	items []*T,
	getID func(*T) *string,
	hasMoreFunc func() ([]*T, error),
	CountFunc func() (int64, error),
) (*PageCursor, error) {
	cursorPage := &PageCursor{}
	if len(items) > 0 {
		cursorPage.FirstID = getID(items[0])
		cursorPage.LastID = getID(items[len(items)-1])
		moreRecords, err := hasMoreFunc()
		if len(moreRecords) > 0 {
			cursorPage.HasMore = true
		}
		if err != nil {
			return nil, err
		}
	}
	count, err := CountFunc()
	if err != nil {
		return cursorPage, err
	}
	cursorPage.Total = count
	return cursorPage, nil
}

func NewCookieWithSecurity(name string, value string, expires time.Time) *http.Cookie {
	// For cross-origin requests (e.g., frontend at different domain), we need SameSite=None with Secure
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	}
}
