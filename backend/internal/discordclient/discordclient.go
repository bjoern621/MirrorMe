// Package discordclient is this app's side of the Discord manager (internal/discordapi).
//
// What Discord mode needs from it: the presence pass that answers channel,
// brokered group and streams in one round trip, and the token trade a command carries.
// Every call reaches a service on another machine,
// so every failure is an Umgebungsfehler and leaves as an error.
// A refusal travels as *groupclient.Refusal,
// the manager refusing in groupd's shape and this side reading both the same way.
package discordclient

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/groupclient"
)

// Timeout bounds every call, as groupclient's does and for the same reason:
// each sits in front of something a user asked for.
const Timeout = 5 * time.Second

// Channel labels where the linked account stands.
type Channel struct {
	Guild string `json:"guild"`
	Name  string `json:"name"`
	// Occupants is everybody sitting in the channel, whatever they run,
	// which is wider than the members the group lists.
	Occupants int `json:"occupants"`
}

// Group is the manager's stand-in for everything a group key would derive locally,
// plus the group and streams as groupd answered them on the same pass.
type Group struct {
	Prefix           string               `json:"prefix"`
	SrtPassphrase    string               `json:"srtPassphrase"`
	MemberID         string               `json:"memberId"`
	DisplayName      string               `json:"displayName"`
	LeaseSeconds     int                  `json:"leaseSeconds"`
	Members          []groupclient.Member `json:"members"`
	PublishingUnread bool                 `json:"publishingUnread"`
	Streams          []groupclient.Stream `json:"streams"`
}

// Answer is one pass's whole truth.
// Channel and Group are nil together, for an account standing in no voice channel.
//
// Application is the Discord application the manager links through, answered on every pass.
// An app states an activity on its own Discord client under that id (internal/discordrpc).
type Answer struct {
	Application string   `json:"application"`
	Channel     *Channel `json:"channel"`
	Group       *Group   `json:"group"`
}

// Client calls the manager. Safe for concurrent use, holding nothing between calls:
// the app's own pass snapshot is the one copy of the last answer.
type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: Timeout}}
}

// Presence states this install's presence and reads the whole answer.
// Idempotent end to end: the manager relays it as groupd's PUT /members.
func (c *Client) Presence(base, linkSecret string) (Answer, error) {
	assert.IsNotNil(c.http, "a client calls through a transport")

	if base == "" {
		return Answer{}, text.Fail(screensharev1.TextCode_TEXT_CODE_RELAY_UNNAMED)
	}

	var answer Answer
	if err := c.send(http.MethodPut, base+"/presence", linkSecret, &answer); err != nil {
		return Answer{}, err
	}
	return answer, nil
}

// Token trades the link secret for a relay access token under the current channel's prefix.
//
// Minted per call and cached nowhere:
// which group it grants follows the voice channel between two calls,
// and a held token would be a copy of a fact only the manager owns.
func (c *Client) Token(base, linkSecret string) (token, prefix string, err error) {
	assert.IsNotNil(c.http, "a client calls through a transport")

	if base == "" {
		return "", "", text.Fail(screensharev1.TextCode_TEXT_CODE_RELAY_UNNAMED)
	}

	var answer struct {
		Token  string `json:"relayAccessToken"`
		Prefix string `json:"prefix"`
	}
	if err := c.send(http.MethodPost, base+"/tokens", linkSecret, &answer); err != nil {
		return "", "", err
	}
	if answer.Token == "" {
		return "", "", text.Fail(screensharev1.TextCode_TEXT_CODE_ADDRESS_ANSWER_UNREADABLE, text.Address(base))
	}
	return answer.Token, answer.Prefix, nil
}

// send makes one call carrying the link secret and decodes its answer,
// or the reason there is none, in groupd's refusal shape.
func (c *Client) send(method, address, linkSecret string, into any) error {
	assert.Assert(address != "", "a call names the route it reaches", method)
	assert.IsNotNil(into, "a call names where its answer is read into", method, address)

	encoded, err := json.Marshal(map[string]string{"linkSecret": linkSecret})
	assert.IsNil(err, "a request body is plain data that always encodes", method, address)
	request, err := http.NewRequest(method, address, strings.NewReader(string(encoded)))
	if err != nil {
		return text.Because(err, screensharev1.TextCode_TEXT_CODE_ADDRESS_UNREADABLE, text.Address(address))
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(request)
	if err != nil {
		return text.Because(err, screensharev1.TextCode_TEXT_CODE_ADDRESS_UNANSWERED, text.Address(address))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var refusal struct {
			Error string `json:"error"`
		}
		reason := resp.Status
		if json.NewDecoder(resp.Body).Decode(&refusal) == nil && refusal.Error != "" {
			reason = refusal.Error
		}
		return text.Because(&groupclient.Refusal{Status: resp.StatusCode, Reason: reason},
			screensharev1.TextCode_TEXT_CODE_ADDRESS_ANSWERED, text.Address(address))
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return text.Because(err, screensharev1.TextCode_TEXT_CODE_ADDRESS_ANSWER_UNREADABLE, text.Address(address))
	}
	return nil
}
