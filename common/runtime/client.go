package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/common/pagination"
)

const (
	defaultPageSize = 100
	RuntimesLimit   = 1000
)

// Client is the interface to interact with the KEB /runtimes API as an HTTP client using OIDC ID token in JWT format.
type Client interface {
	ListRuntimes(params ListParameters) (RuntimesPage, error)
}

type client struct {
	url        string
	httpClient *http.Client
}

// NewClient constructs and returns new Client for KEB /runtimes API
// It takes the following arguments:
//   - url        : base url of all KEB APIs, e.g. https://kyma-env-broker.kyma.local
//   - httpClient : underlying HTTP client used for API call to KEB
func NewClient(url string, httpClient *http.Client) Client {
	return &client{
		url:        url,
		httpClient: httpClient,
	}
}

// ListRuntimes fetches the runtimes from KEB according to the given parameters.
// If params.Page or params.PageSize is not set (zero), the client will fetch and return all runtimes.
func (c *client) ListRuntimes(params ListParameters) (RuntimesPage, error) {
	runtimes := RuntimesPage{}
	getAll := false
	fetchedAll := false
	limitRuntimes := false
	if params.Page == 0 || params.PageSize == 0 {
		getAll = true
		params.Page = 1
		params.PageSize = defaultPageSize
	}

	for _, state := range params.States {
		if state == StateDeprovisioned {
			limitRuntimes = true
			break
		}
	}

	for !fetchedAll {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/runtimes", c.url), nil)
		if err != nil {
			return runtimes, fmt.Errorf("while creating request: %w", err)
		}
		setQuery(req.URL, params)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return runtimes, fmt.Errorf("while calling %s: %w", req.URL.String(), err)
		}

		// Drain response body and close, return error to context if there isn't any.
		defer func() {
			derr := drainResponseBody(resp.Body)
			if err == nil {
				err = derr
			}
			cerr := resp.Body.Close()
			if err == nil {
				err = cerr
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return runtimes, fmt.Errorf("calling %s returned %d (%s) status", req.URL.String(), resp.StatusCode, resp.Status)
		}

		var rp RuntimesPage
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&rp)
		if err != nil {
			return runtimes, fmt.Errorf("while decoding response body: %w", err)
		}

		runtimes.TotalCount = rp.TotalCount
		runtimes.Count += rp.Count
		runtimes.Data = append(runtimes.Data, rp.Data...)
		if getAll {
			params.Page++
			fetchedAll = runtimes.Count >= runtimes.TotalCount
		} else {
			fetchedAll = true
		}

		if limitRuntimes && runtimes.Count >= RuntimesLimit {
			runtimes.Data = runtimes.Data[:RuntimesLimit]
			runtimes.Count = RuntimesLimit
			fetchedAll = true
		}
	}

	return runtimes, nil
}

func setQuery(url *url.URL, params ListParameters) {
	query := url.Query()
	query.Add(pagination.PageParam, strconv.Itoa(params.Page))
	query.Add(pagination.PageSizeParam, strconv.Itoa(params.PageSize))
	if params.OperationDetail != "" {
		query.Add(OperationDetailParam, string(params.OperationDetail))
	}
	if params.KymaConfig {
		query.Add(KymaConfigParam, "true")
	}
	if params.ClusterConfig {
		query.Add(ClusterConfigParam, "true")
	}
	if params.GardenerConfig {
		query.Add(GardenerConfigParam, "true")
	}
	if params.RuntimeResourceConfig {
		query.Add(RuntimeConfigParam, "true")
	}
	if params.Expired {
		query.Add(ExpiredParam, "true")
	}
	if params.WithBindings {
		query.Add(WithBindingsParam, "true")
	}
	if params.Bindings {
		query.Add(BindingsParam, "true")
	}
	if params.Actions {
		query.Add(ActionsParam, "true")
	}
	setParamList(query, GlobalAccountIDParam, params.GlobalAccountIDs)
	setParamList(query, SubAccountIDParam, params.SubAccountIDs)
	setParamList(query, InstanceIDParam, params.InstanceIDs)
	setParamList(query, RuntimeIDParam, params.RuntimeIDs)
	setParamList(query, RegionParam, params.Regions)
	setParamList(query, ShootParam, params.Shoots)
	setParamList(query, PlanParam, params.Plans)
	for _, s := range params.States {
		query.Add(StateParam, string(s))
	}
	url.RawQuery = query.Encode()
}

func setParamList(query url.Values, key string, values []string) {
	for _, value := range values {
		query.Add(key, value)
	}
}

func drainResponseBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}
