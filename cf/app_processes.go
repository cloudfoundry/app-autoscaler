package cf

import (
	"fmt"
	url "net/url"
	"strconv"
	"strings"
)

const (
	ProcessTypeWeb    = "web"
	ProcessTypeWorker = "worker"
)

/*GetAppProcesses
 * Get the processes information for a specific app
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
 */
func (c *Client) GetAppProcesses(appGuid Guid, processTypes ...string) (Processes, error) {

	query := url.Values{"per_page": {strconv.Itoa(c.conf.PerPage)}, "types": {strings.Join(processTypes, ",")}}
	aUrl := fmt.Sprintf("/v3/apps/%s/processes?%s", appGuid, query.Encode())
	pages, err := PagedResourceRetriever[Process]{AuthenticatedClient{c}}.GetAllPages(aUrl)
	if err != nil {
		return nil, fmt.Errorf("failed GetAppProcesses '%s': %w", appGuid, err)
	}
	return pages, nil
}
