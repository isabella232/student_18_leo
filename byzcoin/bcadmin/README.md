# bcadmin - the CLI to configure ByzCoin ledgers

## Create a new ByzCoin, saving the config

```
$ bcadmin create -roster roster.toml
```

The `roster.toml` file is a list of servers what form the cothority that will
maintain the ledger. After running `run_conode.sh local 3` for example, the file
`public.toml` will have the 3 conodes in it. For a larger production deployment,
you will construct the `roster.toml` file by collecting the `public.toml` files
from each of the servers.

The ByzCoin config info (the skipchain ID and the roster for the cothority)
are stored in the local config directory (~/.config/bcadmin or ~/Library/Application
Support/bcadmin) and the filename is printed on stdout. The ByzCoin config file
will be used by other tools to know where to send their transactions.

The secret key is saved in a file named after the public key. It must not be
shared!

To see the config you just made, use `bcadmin show -bc $file`.

## Granting access to contracts

The user who wants to use ByzCoin generates a private key and shares the
public key with you, the ByzCoin admin. You grant access to a given contract
for instructions signed by the given secret key like this:

```
$ bcadmin add -bc $file spawn:eventlog -identity ed25519:dd6419b01b49e3ffd18696c93884dc244b4688d95f55d6c2a4639f2b0ce40710
```

Using the ByzCoin config file you give them and their private key to sign
transactions, they will now be able to use their application to send
transactions.

## Environmnet variables

You can set the environment variable BC to the config file for the ByzCoin
you are currently working with. (Client apps should follow this same standard.)

## Generating a new keypair

```
$ bcadmin key
```

Generates a new keypair and prints the result in the console

Optional flags:

-save file.txt            Outputs the key in file.txt instead of stdin
## Adding a new darc

```
$ bcadmin darc_add -bc $file
```

Adds a new darc with a random keypair for both signing and evolving it.

Optional flags:

-out_id file.txt          Outputs the ID of the darc in file.txt

-out_key file.txt         Outputs the key of the darc in file.txt

-out_desc file.txt        Outputs the full description of the darc in file.txt

-owner key:%x             Creates the darc the the mentioned key as owner (sign & evolve)

-darc darc:%x             Creates the darc using the mentioned darc for creation (uses Genedis node by default)

-sign key:%x              Uses this key to sign the transaction (mandatory if -darc is used)

## Showing a darc

```
$ bcadmin darc_show -bc $file -id darc:%x
```

Shows the darc for which BaseID is the one specified after the -id flags

Optional flags:

-save file.txt            Outputs the description of the darc in file.txt instead of stdin

## Editing the rules of a darc

```
$ bcadmin darc_rule_add $action -bc $file -identity $expr -darc darc:%x -signer key:%x
```

Adds a new rule to the mentioned darcs

```
$ bcadmin darc_rule_update $action -bc $file -identity $expr -darc darc:%x -signer key:%x
```

Updates the said rule in the mentioned darc

```
$ bcadmin darc_rule_delete $action -bc $file -darc darc:%x -signer key:%x 
```

Deletes the said rule in the mentioned darc
