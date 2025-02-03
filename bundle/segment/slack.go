package segment

import "os"

// Sometimes laptop usernames don't correspond to slack usernames. In this case, add a mapping from the former to
// the latter here
var usernameToSlack = map[string]string{
	"jonathangabe":  "jgabe",
	"blaynemoseley": "bmoseley",
}

func GetSlackUserFromEnv() string {
	var username string
	if os.Getenv("BUILDKITE") != "" {
		username = "buildkite"
	} else {
		envUser := os.Getenv("USER")
		corrected, ok := usernameToSlack[envUser]

		if ok {
			username = "@" + corrected
		} else {
			username = "@" + envUser
		}

		if username == "@" {
			username = "Unknown user"
		}
	}

	return username
}
