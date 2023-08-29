package cli

import (
	"fmt"
	"os"
	"strings"

	nexec "github.com/numtide/nits/pkg/exec"
	nutil "github.com/numtide/nits/pkg/nats"

	"github.com/numtide/nits/pkg/subject"

	"github.com/juju/errors"
	"golang.org/x/crypto/ssh"
)

type addAgentCmd struct {
	Cluster        string `help:"Name of the account under which Agents will run"`
	Name           string `help:"A name for the agent account"`
	PublicKey      string `required:"" xor:"key"`
	PublicKeyFile  string `required:"" type:"existingfile" xor:"key"`
	PrivateKeyFile string `required:"" type:"existingfile" xor:"key"`
}

func (a *addAgentCmd) Run() (err error) {
	var nkey string

	if a.PrivateKeyFile != "" {
		var signer ssh.Signer
		if signer, err = nutil.NewSigner(a.PrivateKeyFile); err != nil {
			return errors.Annotate(err, "failed to parse private key file")
		} else if nkey, err = nutil.NKeyForSigner(signer); err != nil {
			return err
		}
	} else {

		var pk ssh.PublicKey
		keyBytes := []byte(a.PublicKey)

		if !(a.PublicKey == "" || strings.Contains(a.PublicKey, "ssh-ed25519")) {
			keyBytes = []byte("ed25519 " + a.PublicKey)
		} else if a.PublicKeyFile != "" {
			if keyBytes, err = os.ReadFile(a.PublicKeyFile); err != nil {
				return errors.Annotate(err, "failed to read public key file")
			}
		}

		if pk, _, _, _, err = ssh.ParseAuthorizedKey(keyBytes); err != nil {
			return errors.Annotate(err, "failed to parse public key")
		}

		nkey, err = nutil.NKeyForPublicKey(pk)
		if err != nil {
			return errors.Annotate(err, "failed to determine nkey for public key")
		}
	}

	agentSubject := fmt.Sprintf("NITS.AGENT.%s.>", nkey)

	return nexec.Sequence(
		nexec.Nsc(
			"add", "mapping", "-a", a.Cluster,
			"--from", subject.AgentWithName(a.Name),
			"--to", subject.AgentService(nkey, "INFO"),
		),
		nexec.Nsc(
			"add", "user", "-a", a.Cluster,
			"-k", nkey,
			"-n", a.Name,
			"--allow-pub", "NITS.CACHE.>",
			"--allow-pubsub", agentSubject,
			"--allow-pub", "$JS.ACK.agent-deployments.>",
			"--allow-pub", "$JS.API.STREAM.NAMES",
			"--allow-pub", "$JS.API.CONSUMER.*.agent-deployments.>",
			"--allow-sub", "$SRV.>",
			"--allow-pub", "_INBOX.>",
		),
	)
}
