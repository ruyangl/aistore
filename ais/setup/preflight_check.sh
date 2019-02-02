#!/bin/bash

# This script is used by Makefile to run preflight-checks.

case $CHECK in
lint)
    echo "Running lint check..." >&2
    err_count=0
    echo "Checking vet..." >&2
    for dir in ${LINT_DIRS}
    do                   
        errs=$(go tool vet "${LINT_VET_IGNORE}" "${dir}" 2>&1 | tee -a /dev/stderr | wc -l)
        err_count=$(($err_count + $errs))
    done
    echo "Checking others..." >&2
    errs=$(${GOPATH}/bin/gometalinter --exclude="${LINT_IGNORE}" --disable-all --enable=golint --enable=errcheck --enable=staticcheck ${LINT_DIRS} 2>&1 | tee -a /dev/stderr | wc -l )
    err_count=$(($err_count + $errs))
    if [ "${err_count}" != "0" ]; then 
        echo "found ${err_count} lint errors, please fix or add exception" >&2
        exit 1
    fi
    exit 0
  ;;
fmt)
    err_count=0
    echo "Running style check..." >&2
    for dir in ${LINT_DIRS}
    do                   
        errs=$(${GOPATH}/bin/goimports -d ${dir} 2>&1 | tee -a /dev/stderr | grep -e "^diff -u" | wc -l)
        err_count=$(($err_count + $errs))
    done
    if [ "${err_count}" != "0" ]; then
        echo "found ${err_count} style errors, run 'make fmt-fix' to fix" >&2
        exit 1
    fi
    exit 0
  ;;
spell)
    err_count=0
    echo "Running spell check..." >&2
    for dir in ${MISSPELL_DIRS}
    do                   
        errs=$(${GOPATH}/bin/misspell "${dir}" 2>&1 | tee -a /dev/stderr | wc -l)
        err_count=$(($err_count + $errs))
    done
    if [ "${err_count}" != "0" ]; then
        echo "found ${err_count} spelling errors, check to confirm, then run 'make spell-fix' to fix" >&2
        exit 1
    fi
    exit 0
  ;;
test-short)
  echo "Running short tests..." >&2
  errs=$(GOCACHE=off BUCKET=${BUCKET} go test -v -p 1 -count 1  -short ../... 2>&1 | tee -a /dev/stderr | grep -e "^--- FAIL" )
  err_count=$(echo ${errs} -n | wc -l)
  if [ "${err_count}" != "0" ]; then
      echo ${errs} >&2
      echo "test-short: ${err_count} failed" >&2
      exit 1
  fi
  exit 0
  ;;
test-long)
  echo "Running long tests..." >&2
  errs=$(GOCACHE=off BUCKET=${BUCKET} go test -v -p 1 -count 1 -timeout 1h ../... 2>&1 | tee -a /dev/stderr | grep -e "^--- FAIL" )
  err_count=$(echo ${errs} -n | wc -l)
  if [ "${err_count}" != "0" ]; then
      echo ${errs} >&2
      echo "test-short: ${err_count} failed" >&2
      exit 1
  fi
  exit 0
  ;;
test-run)
  echo "Running test with regex..." >&2
  errs=$(GOCACHE=off BUCKET=${BUCKET} go test -v -p 1 -count 1 -timeout 1h -run="${RE}" ../... 2>&1 | tee -a /dev/stderr | grep -e "^--- FAIL" )
  err_count=$(echo ${errs} -n | wc -l)
  if [ "${err_count}" != "0" ]; then
      echo ${errs} >&2
      echo "test-run: ${err_count} failed" >&2
      exit 1
  fi
  exit 0
  ;;
"")
  echo "missing environment variable: CHECK=\"checkname\""
  exit 1
  ;;
*)
  echo "unsupported check: $CHECK"
  exit 1
  ;;
esac

echo $Message | mail -s "disk report `date`" anny