package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	//"os"
)

const BOT_AUTH string = "bot secret"


var PUBLIC_KEY *[]byte
var CLIENT *http.Client

//See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-data
//ps this might not be everything, I got bored making new structs (resolved isnt there)
type InteractionData struct {
	id        uint64       `json:"id"`
	name      string       `json:"name"`
	type_     int          `json:"type"`
	options   []OptionData `json:"options"`
	guild_id  uint64       `json:"guild_id"`
	target_id uint64       `json:"target_id"`
}

//yep defiantly not everything here lol - ive bascially given up on useless stuff now\
// See https://discord.com/developers/docs/resources/user#user-object-user-structure
type User struct {
	id            uint64 `json:"id"`
	username      string `json:"username"`
	discriminator string `json:"discriminator"`
	flags         int    `json:"flags"`
	bot           bool   `json:"bot"`
	system        bool   `json:"system"`
}

//Again ive skipped useless stuff
type GuildMember struct {
	user        User     `json:"user"`
	nick        string   `json:"nick"`
	avatar      string   `json:"avatar"`
	roles       []uint64 `json:"roles"`
	permissions string   `json:"permissions"`
}

//holy shit ive just seen how long this is
//nah fuck that
//See https://discord.com/developers/docs/resources/channel#message-object-message-structure
type Message struct {
	id      uint64 `json:"id"`
	content string `json:"content"`
}

//See https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-interaction-data-option-structure
type OptionData struct {
	name    string       `json:"name"`
	type_   int          `json:"type"`
	value   string       `json:"value"`   //ok this actually is scary cause we dont know the data type of what it is (panic I guess) (ok update gonna make it string)
	options []OptionData `json:"options"` //confused by this ngl
	focused bool         `json:"focused"`
}

const (
	PING                             int = 0
	APPLICATION_COMMAND              int = 2
	MESSAGE_COMPONENT                int = 3
	APPLICATION_COMMAND_AUTOCOMPLETE int = 4
	MODAL_SUBMIT                     int = 5
)

const (
	PONG                                    int = 1
	CHANNEL_MESSAGE_WITH_SOURCE             int = 4
	DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE    int = 5
	DEFERRED_UPDATE_MESSAGE                 int = 6
	UPDATE_MESSAGE                          int = 7
	APPLICATION_COMMAND_AUTOCOMPLETE_RESULT int = 8
	MODAL                                   int = 9
)

// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	id              uint64          `json:"id"`
	application_id  uint64          `json:"application_id"`
	type_           int             `json:"type"`
	data            InteractionData `json:"data"`
	guild_id        uint64          `json:"guild_id"`
	channel_id      uint64          `json:"channel_id"`
	member          GuildMember     `json:"member"`
	user            User            `json:"user"`
	token           string          `json:"token"`
	version         int             `json:"version"`
	message         Message         `json:"message"`
	app_permissions string          `json:"app_permissions"`
	locale          string          `json:"locale"`
	guild_locale    string          `json:"guild_locale"`
}

type Error struct {
	code    int    `json:"code"`
	message string `json:"message"`
}

func makeForumChannel(guild *uint64, name *string, responsible *string) error {
	postBody, _ := json.Marshal(map[string]string{
		"name": *name,
		"type": "13",
	})
	responseBody := bytes.NewBuffer(postBody)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://discord.com/api/v10/guilds/%d/channels", *guild), responseBody)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bot "+BOT_AUTH)
	req.Header.Add("X-Audit-Log-Reason", *responsible+" told me to.")
	req.Header.Set("Content-Type", "application/json")

	resp, err := CLIENT.Do(req)
	if err != nil {
		return err
	}
	fmt.Println(resp.StatusCode)

	if resp.StatusCode != 201 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		errorReason := Error{}
		err = json.Unmarshal(b, &errorReason)
		if err != nil {
			panic(err)
		}
		return errors.New(errorReason.message)
	}
	return nil
}

func errorInternal(w *http.ResponseWriter) {
	(*w).WriteHeader(500)
	fmt.Println("internal error")
}

func errorUnauth(w *http.ResponseWriter) {
	(*w).WriteHeader(401)
	fmt.Println("unauthorised")
}

func checkIfVerified(hexPublicKey *[]byte, timestamp *string, b *[]byte, signiture *string) bool {
	sig, err := hex.DecodeString(*signiture)
	if err != nil {
		panic(err)
	}
	return ed25519.Verify([]byte(*hexPublicKey), append([]byte(*timestamp), *b...), sig)
}

func ThouHasBeenContacted(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	signiture := r.Header.Get("X-Signature-Ed25519")
	rawTime := r.Header.Get("X-Signature-Timestamp")

	if signiture == "" || rawTime == "" {
		errorUnauth(&w)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorInternal(&w)
		panic(err)
	}

	if !checkIfVerified(PUBLIC_KEY, &rawTime, &b, &signiture) {
		errorUnauth(&w)
		return
	}

	interaction := Interaction{}
	err = json.Unmarshal(b, &interaction)
	if err != nil {
		panic(err)
	}

	if interaction.type_ == PING {
		w.WriteHeader(200)
		fmt.Fprintln(w, "{type: 1}")
	} else if interaction.type_ == APPLICATION_COMMAND {
		name := string(interaction.data.options[0].value) //name
		guildId := interaction.guild_id
		err = makeForumChannel(&guildId, &name, &interaction.member.user.username)
		if err != nil {
			panic(err)
		}

	}
}

//runs when new google cloud instance thing starts
func main() {
	e, err := hex.DecodeString("public key")
	if err != nil {
		panic(err)
	}
	PUBLIC_KEY = &e
	CLIENT = &http.Client{}

	http.HandleFunc("/", ThouHasBeenContacted)
	http.ListenAndServe(":5000", nil)
}
