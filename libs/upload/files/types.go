package files

import "errors"

// Sentinel errors surfaced by the Files API client.
var (
	ErrUnexpectedServerResponse = errors.New("unexpected server response")
	ErrAlreadyExists            = errors.New("the file being created already exists")
)

// --- Exported result types ---

// InitiateResult reports which large-file protocol the server selected. Exactly
// one of MultipartUpload (AWS/Azure) and ResumableUpload (GCP) is non-nil on a
// well-formed response; the pointers distinguish "not offered" from "offered
// with an empty token".
type InitiateResult struct {
	MultipartUpload *UploadSession `json:"multipart_upload"`
	ResumableUpload *UploadSession `json:"resumable_upload"`
}

type UploadSession struct {
	SessionToken string `json:"session_token"`
}

// PresignedURL is a resolved cloud-storage URL with its associated request
// headers, as minted by the control plane and transferred over by the caller.
type PresignedURL struct {
	URL        string
	PartNumber int
	Headers    map[string]string
}

// --- Wire types for the Files API multipart/resumable coordination ---

type nameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type createPartURLsRequest struct {
	Path            string `json:"path"`
	SessionToken    string `json:"session_token"`
	StartPartNumber int    `json:"start_part_number"`
	Count           int    `json:"count"`
	ExpireTime      string `json:"expire_time"`
}

type createPartURLsResponse struct {
	UploadPartURLs []struct {
		URL        string      `json:"url"`
		PartNumber int         `json:"part_number"`
		Headers    []nameValue `json:"headers"`
	} `json:"upload_part_urls"`
}

type resumableURLRequest struct {
	Path         string `json:"path"`
	SessionToken string `json:"session_token"`
}

// urlWithHeaders is a presigned cloud-storage URL and its required request
// headers, as returned by the resumable and abort URL endpoints.
type urlWithHeaders struct {
	URL     string      `json:"url"`
	Headers []nameValue `json:"headers"`
}

type resumableURLResponse struct {
	ResumableUploadURL *urlWithHeaders `json:"resumable_upload_url"`
}

type abortURLRequest struct {
	Path         string `json:"path"`
	SessionToken string `json:"session_token"`
	ExpireTime   string `json:"expire_time"`
}

type abortURLResponse struct {
	AbortUploadURL *urlWithHeaders `json:"abort_upload_url"`
}

type completePart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type completeRequest struct {
	Parts []completePart `json:"parts"`
}
