package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/byzcoin/bcadmin/lib"
	"github.com/dedis/cothority/byzcoin/darc"
	"github.com/dedis/onet"
	"github.com/dedis/onet/app"
	"github.com/dedis/onet/cfgpath"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
	cli "gopkg.in/urfave/cli.v1"
)



func init() {
	network.RegisterMessages(&darc.Darc{}, &darc.Identity{}, &darc.Signer{})
}

var cmds = cli.Commands{
	{
		Name:    "create",
		Usage:   "create a ledger",
		Aliases: []string{"c"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "roster, r",
				Usage: "the roster of the cothority that will host the ledger",
			},
			cli.DurationFlag{
				Name:  "interval, i",
				Usage: "the block interval for this ledger",
				Value: 5 * time.Second,
			},
		},
		Action: create,
	},
	{
		Name:    "show",
		Usage:   "show the config, contact ByzCoin to get Genesis Darc ID",
		Aliases: []string{"s"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
		},
		Action: show,
	},
	{
		Name:    "add",
		Usage:   "add a rule and signer to the base darc",
		Aliases: []string{"a"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "identity",
				Usage: "the identity of the signer who will be allowed to access the contract (e.g. ed25519:a35020c70b8d735...0357))",
			},
		},
		Action: add,
	},
	{
		Name:    "key",
		Usage:   "generates a new keypair and prints the public key in the stdin",
		Aliases: []string{"k"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "save",
				Usage: "file in which the user wants to save the public key instead of printing it",
			},
		},
		Action: key,
	},
	{
		Name:    "darc_add",
		Usage:   "add a new darc to the instance",
		Aliases: []string{"da"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "owner",
				Usage: "owner of the darc allowed to sign and evolve it",
			},
			cli.StringFlag{
				Name:  "darc",
				Usage: "darc from which we create the new darc - genesis if not mentioned",
			},
			cli.StringFlag{
				Name:  "sign",
				Usage: "signature for the generating darc",
			},
			cli.StringFlag{
				Name:  "out_id",
				Usage: "output file for the darc id",
			},
			cli.StringFlag{
				Name:  "out_desc",
				Usage: "output file for the darc description",
			},
			cli.StringFlag{
				Name:  "out_key",
				Usage: "output file for the darc key",
			},
		},
		Action: darc_add,
	},
	{
		Name:    "darc_show",
		Usage:   "shows darc with given ID",
		Aliases: []string{"ds"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "id",
				Usage: "the id of the darc one wants to show",
			},
			cli.StringFlag{
				Name:  "save",
				Usage: "file in which the user wants to save the darc",
			},
		},
		Action: darc_show,
	},
	{
		Name:    "darc_rule_add",
		Usage:   "add a rule and signer to a darc",
		Aliases: []string{"dra"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "identity",
				Usage: "the identity of the signer who will be allowed to access the contract (e.g. ed25519:a35020c70b8d735...0357))",
			},
			cli.StringFlag{
				Name:  "darc",
				Usage: "the darc to which we want to add a rule and signer",
			},
			cli.StringFlag{
				Name:  "signer",
				Usage: "the signer of the operation",
			},
		},
		Action: darc_rule_add,
	},
	{
		Name:    "darc_rule_update",
		Usage:   "update the signers for the given rule",
		Aliases: []string{"dru"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "identity",
				Usage: "the identity of the signer who will be allowed to access the contract (e.g. ed25519:a35020c70b8d735...0357))",
			},
			cli.StringFlag{
				Name:  "darc",
				Usage: "the darc to which we want to delete",
			},
			cli.StringFlag{
				Name:  "signer",
				Usage: "the signer of the operation",
			},
		},
		Action: darc_rule_update,
	},
	{
		Name:    "darc_rule_del",
		Usage:   "deletes a rule of the darc",
		Aliases: []string{"drd"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use",
			},
			cli.StringFlag{
				Name:  "darc",
				Usage: "the darc to which we want to delete",
			},
			cli.StringFlag{
				Name:  "signer",
				Usage: "the signer of the operation",
			},
		},
		Action: darc_rule_del,
	},
}

var cliApp = cli.NewApp()

// getDataPath is a function pointer so that tests can hook and modify this.
var getDataPath = cfgpath.GetDataPath

func init() {
	cliApp.Name = "bc"
	cliApp.Usage = "Create ByzCoin ledgers and grant access to them."
	cliApp.Version = "0.1"
	cliApp.Commands = cmds
	cliApp.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
		cli.StringFlag{
			Name:  "config, c",
			Value: "",
			Usage: "path to configuration-directory",
		},
	}
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		lib.ConfigPath = c.String("config")
		if lib.ConfigPath == "" {
			lib.ConfigPath = getDataPath(cliApp.Name)
		}
		return nil
	}
}

func main() {
	log.ErrFatal(cliApp.Run(os.Args))
}

func create(c *cli.Context) error {
	fn := c.String("roster")
	if fn == "" {
		return errors.New("--roster flag is required")
	}

	in, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("Could not open roster %v: %v", fn, err)
	}
	r, err := readRoster(in)
	if err != nil {
		return err
	}

	interval := c.Duration("interval")

	owner := darc.NewSignerEd25519(nil, nil)

	req, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:darc"}, owner.Identity())
	if err != nil {
		return err
	}
	req.BlockInterval = interval

	cl := onet.NewClient(cothority.Suite, byzcoin.ServiceName)

	var resp byzcoin.CreateGenesisBlockResponse
	err = cl.SendProtobuf(r.List[0], req, &resp)
	if err != nil {
		return err
	}

	cfg := lib.Config{
		ByzCoinID:     resp.Skipblock.SkipChainID(),
		Roster:        *r,
		GenesisDarc:   req.GenesisDarc,
		AdminIdentity: owner.Identity(),
	}
	fn, err = lib.SaveConfig(cfg)
	if err != nil {
		return err
	}

	err = lib.SaveKey(owner)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Created ByzCoin with ID %x.\n", cfg.ByzCoinID)
	fmt.Fprintf(c.App.Writer, "export BC=\"%v\"\n", fn)

	// For the tests to use.
	c.App.Metadata["BC"] = fn

	return nil
}

func show(c *cli.Context) error {
	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.App.Writer, "ByzCoinID:", fmt.Sprintf("%x", cfg.ByzCoinID))
	fmt.Fprintln(c.App.Writer, "Genesis Darc:")
	var roster []string
	for _, s := range cfg.Roster.List {
		roster = append(roster, string(s.Address))
	}
	fmt.Fprintln(c.App.Writer, "Roster:", strings.Join(roster, ", "))

	gd, err := cl.GetGenDarc()
	if err == nil {
		fmt.Fprintln(c.App.Writer, gd)
	} else {
		fmt.Fprintln(c.App.ErrWriter, "could not fetch darc:", err)
	}

	return err
}

func add(c *cli.Context) error {
	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	signer, err := lib.LoadKey(cfg.AdminIdentity)
	if err != nil {
		return err
	}

	arg := c.Args()
	if len(arg) == 0 {
		return errors.New("need the rule to add, e.g. spawn:contractName")
	}
	action := arg[0]

	identity := c.String("identity")
	if identity == "" {
		return errors.New("--identity flag is required")
	}

	d, err := cl.GetGenDarc()
	if err != nil {
		return err
	}

	d2 := d.Copy()
	d2.EvolveFrom(d)

	d2.Rules.AddRule(darc.Action(action), []byte(identity))

	d2Buf, err := d2.ToProto()
	if err != nil {
		return err
	}

	invoke := byzcoin.Invoke{
		Command: "evolve",
		Args: []byzcoin.Argument{
			byzcoin.Argument{
				Name:  "darc",
				Value: d2Buf,
			},
		},
	}
	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(d2.GetBaseID()),
		Index:      0,
		Length:     1,
		Invoke:     &invoke,
		Signatures: []darc.Signature{
			darc.Signature{Signer: signer.Identity()},
		},
	}
	err = instr.SignBy(d2.GetBaseID(), *signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{instr},
	}, 10)
	if err != nil {
		return err
	}

	return nil
}

func key(c *cli.Context) error {
	newSigner := darc.NewSignerEd25519(nil, nil)
	lib.SaveKey(newSigner)

	save := c.String("save")
	if save == "" {
		fmt.Println(newSigner.Identity().String())
	} else {
		fo, err := os.Create(save)
	  if err != nil {
	      return err
	  }

		fo.Write([]byte(newSigner.Identity().String()))

		fo.Close()
	}

	return nil
}

func darc_add(c *cli.Context) error {
	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	var signer *darc.Signer
	var dGen *darc.Darc

	dstr := c.String("darc")
	if dstr == "" {
		signer, err = lib.LoadKey(cfg.AdminIdentity)
		if err != nil {
			return err
		}

		dGen, err = cl.GetGenDarc()
		if err != nil {
			return err
		}
	} else {
		sstr := c.String("sign")
		if bcArg == "" {
			return errors.New("--sign flag is required if --darc flag is used")
		}

		signer, err = lib.LoadKeyFromString(sstr)
		if err != nil {
			return err
		}

		dGen, err = getDarcByString(cl, dstr)
		if err != nil {
			return err
		}
	}

	var identity darc.Identity
	var newSigner darc.Signer

	owner := c.String("owner")
	if owner != "" {
		tmpSigner, err := lib.LoadKeyFromString(owner)
		if err != nil {
			return err
		}
		newSigner = *tmpSigner
		identity = newSigner.Identity()
	} else {
		newSigner = darc.NewSignerEd25519(nil, nil)
		lib.SaveKey(newSigner)
		identity = newSigner.Identity()
	}

	rules := darc.InitRulesWith([]darc.Identity{identity}, []darc.Identity{identity}, "invoke:evolve")
	d := darc.NewDarc(rules, nil)

	_, err = getDarcByID(cl, d.GetBaseID())
	if err == nil {
		return errors.New("Cannot create a darc with the same BaseID as one that already exists\nPlease check that there isn't a darc with the exat same owner")
	}

	dBuf, err := d.ToProto()
	if err != nil {
		return err
	}

	instID := byzcoin.NewInstanceID(dGen.GetBaseID())

	spawn := byzcoin.Spawn{
		ContractID: "darc",
		Args: []byzcoin.Argument{
			byzcoin.Argument{
				Name:  "darc",
				Value: dBuf,
			},
		},
	}
	instr := byzcoin.Instruction{
		InstanceID: instID,
		Index:      0,
		Length:     1,
		Spawn:     &spawn,
		Signatures: []darc.Signature{
			darc.Signature{Signer: signer.Identity()},
		},
	}
	err = instr.SignBy(dGen.GetBaseID(), *signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{instr},
	}, 10)
	if err != nil {
		return err
	}

	fmt.Println(d.String())

	/*Saving ID in special file*/

	output := c.String("out_id")
	if output != "" {
		fo, err := os.Create(output)
	  if err != nil {
	      panic(err)
	  }

		fo.Write([]byte(d.GetIdentityString()))

		fo.Close()
	}

	/*Saving key in special file*/

	output = c.String("out_key")
	if output != "" {
		fo, err := os.Create(output)
	  if err != nil {
	      panic(err)
	  }

		fo.Write([]byte(newSigner.Identity().String()))

		fo.Close()
	}

	/*Saving description in special file*/

	output = c.String("out_desc")
	if output != "" {
		fo, err := os.Create(output)
	  if err != nil {
	      panic(err)
	  }

		fo.Write([]byte(d.String()))

		fo.Close()
	}

	return nil
}

func darc_show(c *cli.Context) error {
	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	_, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	id := c.String("id")
	if id == "" {
		return errors.New("--id flag is required")
	}

	d, err := getDarcByString(cl, id)
	if err != nil {
		return err
	}

	/*Saving description in special file*/

	output := c.String("save")
	if output != "" {
		fo, err := os.Create(output)
	  if err != nil {
	      panic(err)
	  }

		fo.Write([]byte(d.String()))

		fo.Close()
	} else {
		fmt.Println(d.String())
	}

	return nil
}

func darc_rule(c *cli.Context, update bool) error {
	var str string
	if update {
		str = "update"
	} else {
		str = "add"
	}

	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	_, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	arg := c.Args()
	if len(arg) == 0 {
		return errors.New("need the rule to " + str + ", e.g. spawn:contractName")
	}
	action := arg[0]

	identity := c.String("identity")
	if identity == "" {
		return errors.New("--identity flag is required")
	}

	dstr := c.String("darc")
	if dstr == "" {
		return errors.New("--darc flag is required")
	}

	sig := c.String("signer")
	if sig == "" {
		return errors.New("--signer flag is required")
	}

	d, err := getDarcByString(cl, dstr)
	if err != nil {
		return err
	}

	d2 := d.Copy()
	d2.EvolveFrom(d)

	if update {
		err = d2.Rules.UpdateRule(darc.Action(action), []byte(identity))
	} else {
		err = d2.Rules.AddRule(darc.Action(action), []byte(identity))
	}

	if err != nil {
		return err
	}

	d2Buf, err := d2.ToProto()
	if err != nil {
		return err
	}

	signer, err := lib.LoadKeyFromString(sig)
	if err != nil {
		return err
	}

	invoke := byzcoin.Invoke{
		Command: "evolve",
		Args: []byzcoin.Argument{
			byzcoin.Argument{
				Name:  "darc",
				Value: d2Buf,
			},
		},
	}
	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(d2.GetBaseID()),
		Index:      0,
		Length:     1,
		Invoke:     &invoke,
		Signatures: []darc.Signature{
			darc.Signature{Signer: signer.Identity()},
		},
	}
	err = instr.SignBy(d2.GetBaseID(), *signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{instr},
	}, 10)
	if err != nil {
		return err
	}

	return nil
}

func darc_rule_add(c *cli.Context) error {
	return darc_rule(c, false)
}

func darc_rule_update(c *cli.Context) error {
	return darc_rule(c, true)
}

func darc_rule_del(c *cli.Context) error {
	bcArg := c.String("bc")
	if bcArg == "" {
		return errors.New("--bc flag is required")
	}

	_, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	arg := c.Args()
	if len(arg) == 0 {
		return errors.New("need the rule to delete, e.g. spawn:contractName")
	}
	action := arg[0]

	dstr := c.String("darc")
	if dstr == "" {
		return errors.New("--darc flag is required")
	}

	sig := c.String("signer")
	if sig == "" {
		return errors.New("--signer flag is required")
	}

	d, err := getDarcByString(cl, dstr)
	if err != nil {
		return err
	}

	d2 := d.Copy()
	d2.EvolveFrom(d)

	err = d2.Rules.DeleteRules(darc.Action(action))
	if err != nil {
		return err
	}

	d2Buf, err := d2.ToProto()
	if err != nil {
		return err
	}

	signer, err := lib.LoadKeyFromString(sig)
	if err != nil {
		return err
	}

	invoke := byzcoin.Invoke{
		Command: "evolve",
		Args: []byzcoin.Argument{
			byzcoin.Argument{
				Name:  "darc",
				Value: d2Buf,
			},
		},
	}
	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(d2.GetBaseID()),
		Index:      0,
		Length:     1,
		Invoke:     &invoke,
		Signatures: []darc.Signature{
			darc.Signature{Signer: signer.Identity()},
		},
	}
	err = instr.SignBy(d2.GetBaseID(), *signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{instr},
	}, 10)
	if err != nil {
		return err
	}

	return nil
}

type configPrivate struct {
	Owner darc.Signer
}

func init() { network.RegisterMessages(&configPrivate{}) }

func readRoster(r io.Reader) (*onet.Roster, error) {
	group, err := app.ReadGroupDescToml(r)
	if err != nil {
		return nil, err
	}

	if len(group.Roster.List) == 0 {
		return nil, errors.New("empty roster")
	}
	return group.Roster, nil
}

func rosterToServers(r *onet.Roster) []network.Address {
	out := make([]network.Address, len(r.List))
	for i := range r.List {
		out[i] = r.List[i].Address
	}
	return out
}

func getDarcByString(cl *byzcoin.Client, id string) (*darc.Darc, error) {
	var xrep []byte
	fmt.Sscanf(id[5:], "%x", &xrep)
	return getDarcByID(cl, xrep)
}

func getDarcByID(cl *byzcoin.Client, id []byte) (*darc.Darc, error) {
	pr, err := cl.GetProof(id)
	if err != nil {
		return nil, err
	}

	p := &pr.Proof
	_, vs, err := p.KeyValue()
	if err != nil {
		return nil, err
	}


	d, err := darc.NewFromProtobuf(vs[0])
	if err != nil {
		return nil, err
	}

	return d, nil
}
