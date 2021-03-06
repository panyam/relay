package goclient

import (
	"errors"
	"fmt"
	authcore "github.com/panyam/relay/services/auth/core"
	msgcore "github.com/panyam/relay/services/msg/core"
	"github.com/panyam/relay/utils"
	"io"
	"log"
	"net/http"
	"strings"
	// "os"
	// "os/exec"
	// "sort"
	// "strconv"
)

type ApiClient struct {
	Url           string
	Authenticator Authenticator
}

func NewApiClient(url string) *ApiClient {
	client := ApiClient{Url: url}
	log.Println("New Api Client, URL: ", url)
	return &client
}

func (client *ApiClient) url(path string) string {
	out := client.Url
	if strings.HasPrefix(path, "/") {
		return out + path
	} else {
		return out + "/" + path
	}
}

func (client *ApiClient) EnableAuthentication(auth Authenticator) {
	client.Authenticator = auth
}

func (client *ApiClient) DisableAuthentication() {
	client.Authenticator = nil
}

func (client *ApiClient) MakeAuthRequest(method, endpoint string,
	queryParams map[string]string, body io.Reader) (*http.Request, error) {
	request, err := MakeRequest(method, client.url(endpoint), queryParams, body)
	if err != nil {
		return nil, err
	}
	if client.Authenticator != nil {
		client.Authenticator.AuthenticateRequest(request)
	}
	return request, nil
}

func (client *ApiClient) RegisterUser(team *msgcore.Team, username string, address_type string, address string, password string) (*authcore.Registration, error) {
	req, err := NewRegisterUserRequest(team, username, address_type, address, password)
	var data map[string]interface{}
	_, err = SendRequest(req, &data)
	if err != nil {
		return nil, err
	}

	return authcore.RegistrationFromDict(data)
}

func (client *ApiClient) ConfirmRegistration(registration *authcore.Registration, verificationCode string) error {
	req, err := NewConfirmRegistrationRequest(registration, verificationCode)
	resp, err := SendRequest(req, nil)
	if err != nil {
		if resp.StatusCode == 400 {
			return errors.New("Confirmation failed")
		} else if resp.StatusCode != 200 {
			return errors.New(resp.Status)
		}
	}
	return err
}

func (client *ApiClient) GetTeams() ([]*msgcore.Team, error) {
	req, err := NewGetTeamsRequest()
	var data map[string]interface{}
	_, err = SendRequest(req, &data)
	return nil, err
}

func (client *ApiClient) CreateTeam(team *Team) (*msgcore.Team, error) {
	req, err := NewCreateTeamsRequest(team)
	resp, err := SendRequest(req, team)
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	return team, err
}

func (client *ApiClient) CreateChannel(request *CreateChannelRequest) (*msgcore.Channel, error) {
	req, err := NewCreateChannelRequest(request)
	var data map[string]interface{}
	resp, err := SendRequest(req, &data)
	log.Println("Response: ", resp)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	return msgcore.ChannelFromDict(data)
}

// Invite users to a channel.
//
// **Endpoints:** POST /channels/<id>/
//
// **Auth Required:** YES and user must be a participant in the channel AND have
// "invite_<channelid>" permission.
//
// **Parameters:**
//
// - participants: Comma seperated list of usernames
// - createusers: Whether to create users who are not registered yet.  Requires
// "createusers" permissions.
//
// **Return:**
//
// HTTP Status 200 on success along with team details, eg:
//
// ```
// {"id": "123", "name": "Dream Team", "organization": "Dream Owner"}
// ```
func (client *ApiClient) InviteToChannel(channel *msgcore.Channel, participants []string) error {
	return nil
}

// Get channels for a user
//
// **Endpoints:** GET /teams/{teamid}/channels/
//
// **Auth Required:** YES
//
// **Parameters:**
//		owner:	Channels with this user as the owner will be returned.
//		participants:	Comma seperate usernames to filter channels by (ie
//						contains these participants)
//		matchall: 		Whether to match ALL participants or ANY
//						participant in the above list
// 		order_by: 		Channel fields to order by (prefixed by - indicates descending order):
//     		* name 				- order by name of the channel
//     		* created 			- order by created date
//     		* updated 			- order by last updated
//     		* last_messaged 	- order by last message date
//
// **Return:**
//
// HTTP Status 200 on success along with list of a channels and their members
// matching the result.  If the owner AND participants parameters are not
// specified then only public channels are returned.
func (client *ApiClient) GetChannels(team *msgcore.Team,
	owner string,
	participants []string,
	matchall bool,
	order_by string) ([]*msgcore.Channel, [][]msgcore.ChannelMember, error) {

	params := map[string]string{"owner": owner,
		"participants": strings.Join(participants, ","),
		"matchall":     "false",
	}
	if matchall {
		params["matchall"] = "true"
	}

	url := fmt.Sprintf("/teams/%s/channels/", utils.ID2String(team.Id))
	req, err := client.MakeAuthRequest("GET", url, params, nil)
	var data map[string]interface{}
	resp, err := SendRequest(req, &data)
	log.Println("GetChannels Response: ", resp)
	log.Println("GetChannels Response Data: ", data)
	if err != nil {
		return nil, nil, err
	} else if resp.StatusCode != 200 {
		return nil, nil, errors.New(resp.Status)
	}
	return nil, nil, nil // msgcore.ChannelFromDict(data)
}
