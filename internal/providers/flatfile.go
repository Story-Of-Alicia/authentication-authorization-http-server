package providers

import (
	"authentication-server/internal"
	"authentication-server/internal/facade"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type FlatFileSessionProvider struct {
	WorkDir string
}

func (s *FlatFileSessionProvider) CreateSession(username string) (facade.Session, error) {
	filename := fmt.Sprintf("%s/%s.json", s.WorkDir, username)
	var profile map[string]interface{}
	_, err := os.Stat(filename)
	if err == nil {
		/* file exists, read existing contents */
		b, err := os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return facade.Session{}, err
		}

		err = json.Unmarshal(b, &profile)
		if err != nil {
			log.Println(err)
			return facade.Session{}, err
		}
	} else if os.IsNotExist(err) {
		/* file doesn't exist, sensible defaults */
		profile = map[string]interface{}{
			"characterUid": 0,
			"infractions":  make([]string, 0),
			"name":         username,
			"token":        "",
		}
	} else {
		/* just an error */
		log.Println(err)
		return facade.Session{}, err
	}

	token, err := internal.GenerateSessionToken(internal.TokenSize)
	if err != nil {
		log.Println(err)
		return facade.Session{}, err
	}

	profile["token"] = token

	buf, err := json.Marshal(profile)
	if err != nil {
		log.Println(err)
		return facade.Session{}, err
	}

	err = os.WriteFile(filename, buf, 0644)
	if err != nil {
		log.Println(err)
		return facade.Session{}, err
	}

	return facade.Session{
		Token:  token,
		User:   username,
		Expiry: time.Now().Add(time.Hour * 1),
	}, nil
}

func (s *FlatFileSessionProvider) DeleteSession(username string) error {
	return nil
}
