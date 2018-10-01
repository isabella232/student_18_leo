#!/usr/bin/env bash

DBG_TEST=0
DBG_SRV=0

. "$(go env GOPATH)/src/github.com/dedis/cothority/libtest.sh"

main(){
    startTest
    buildConode github.com/dedis/cothority/byzcoin
    run testCreateStoreRead
    run testAddDarc
    run testRuleDarc
    run testAddDarcFromOtherOne
    run testAddDarcWithOwner
    run testExpression
    stopTest
}

testCreateStoreRead(){
	runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
	eval $SED
	[ -z "$BC" ] && exit 1
    testOK ./"$APP" add spawn:xxx -identity ed25519:foo
	testGrep "ed25519:foo" ./"$APP" show
}

testAddDarc(){
  runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
  eval $SED
  [ -z "$BC" ] && exit 1

  echo -e "\n\nTESTING ADDITION\n"
  testOK ./"$APP" darc_add
  testOK ./"$APP" darc_add -out_id ./darc_id.txt
  testOK ./"$APP" darc_add
  ID=`cat ./darc_id.txt`
  echo -e "\n\nTESTING SHOW\n"
  testOK ./"$APP" darc_show --id "$ID"
}

testRuleDarc(){
  runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
  eval $SED
  [ -z "$BC" ] && exit 1

  testOK ./"$APP" darc_add -out_id ./darc_id.txt -out_key ./darc_key.txt
  ID=`cat ./darc_id.txt`
  KEY=`cat ./darc_key.txt`
  testOK ./"$APP" darc_rule_add spawn:xxx -identity ed25519:foo -darc "$ID" -signer "$KEY"
  testOK ./"$APP" darc_show -id "$ID"
  testOK ./"$APP" darc_rule_update spawn:xxx -identity "ed25519:foo | ed25519:oof" -darc "$ID" -signer "$KEY"
  testOK ./"$APP" darc_show -id "$ID"
  testOK ./"$APP" darc_rule_del spawn:xxx -darc "$ID" -signer "$KEY"
  testOK ./"$APP" darc_show -id "$ID"
}

testAddDarcFromOtherOne(){
  runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
  eval $SED
  [ -z "$BC" ] && exit 1

  testOK ./"$APP" darc_add -out_key ./key.txt -out_id ./id.txt
  KEY=`cat ./key.txt`
  ID=`cat ./id.txt`
  testOK ./"$APP" darc_rule_add spawn:darc -identity "$KEY" -darc "$ID" -signer "$KEY"
  testOK ./"$APP" darc_add -darc "$ID" -sign "$KEY"
  testOK ./"$APP" darc_show -id "$ID"
}

testAddDarcWithOwner(){
  runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
  eval $SED
  [ -z "$BC" ] && exit 1

  testOK ./"$APP" key -save ./key.txt
  KEY=`cat ./key.txt`
  echo "Generated key: $KEY"
  testOK ./"$APP" darc_add -owner "$KEY"
}

testExpression(){
  runCoBG 1 2 3
    runGrepSed "export BC=" "" ./"$APP" create --roster public.toml --interval .5s
  eval $SED
  [ -z "$BC" ] && exit 1

  testOK ./"$APP" darc_add -out_id ./darc_id.txt -out_key ./darc_key.txt
  ID=`cat ./darc_id.txt`
  KEY=`cat ./darc_key.txt`
  testOK ./"$APP" key -save ./key.txt
  KEY2=`cat ./key.txt`

  testOK ./"$APP" darc_rule_add spawn:darc -identity "$KEY | $KEY2" -darc "$ID" -signer "$KEY"
  testOK ./"$APP" darc_show -id "$ID"
  testOK ./"$APP" darc_add -darc "$ID" -sign "$KEY"
  testOK ./"$APP" darc_add -darc "$ID" -sign "$KEY2"
}

main
