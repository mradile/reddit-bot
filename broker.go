package mswkn

const (
	BrokerSubjectWKNRequest          = "request.wkns"
	BrokerSubjectSecuritiesRequest   = "request.securities"
	BrokerSubjectInfoLinksRequest    = "request.infolinks"
	BrokerSubjectRedditRepplyRequest = "request.redditreply"
)

type RedditRequest struct {
	//Name is an ID of the reddit comment
	Name string
	//Text is the comment body
	Text string
}

type SecuritiesRequest struct {
	//Name is an ID of the reddit comment
	Name string
	//WKNs requested from user
	WKNs []string
}

type InfoLinksRequest struct {
	//Name is an ID of the reddit comment
	Name string
	//WKNs requested from user
	WKNs []string
	//Contains a map with WKNs as keys and the found Security list
	Securities map[string]*Security
	//Contains a map with WKNs as keys and the errors
	ResponseErrors map[string]error
}

type RedditReplyRequest struct {
	//Name is an ID of the reddit comment
	Name string
	//WKNs requested from user
	WKNs []string
	//Securities is a map with WKNs as keys and the found Security list
	Securities map[string]*Security
	//InfoLinks is a map with WKNs as keys and the found InfoLink list
	InfoLinks map[string]*InfoLink
}

type Broker interface {
	Publish(subject string, v interface{}) error
	Subscribe(subject string, handler interface{}) error
	Close()
}
