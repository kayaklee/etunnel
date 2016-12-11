#!/bin/sh

self=$(cd `dirname $0`; pwd)
export GOPATH=$self

author="yubai@oceanbase.org.cn"
gitbranch=`git status | head -n1 | awk '{print $NF}'`
githash=`git rev-parse HEAD`
githash="$githash@$gitbranch"
buildstamp=`date '+%Y-%m-%d_%I:%M:%S%p'`

build() {
  go build -o $self/build/bin/$1 -v -i -ldflags "
    -X common.author=$author
    -X common.githash=$githash
    -X common.buildstamp=$buildstamp" apps/$1

  mkdir -p $self/build/etc \
      && cp $self/src/etc/* $self/build/etc/
}

check() {
  go tool vet -all --composites=false -shadowstrict $self/src
}

clean() {
  rm -rf $self/build
  rm -rf $self/pkg
}

gotest() {
  pkg=`grep package $1 | awk '{print $NF}'`
  path=`find $self/src -name "$pkg"`
  pkg=`echo $path | awk -v "b=$self/src/" '{print substr($1, length(b) + 1)}'`
  cases=""
  for t in `grep "func Test.*(t \*testing\.T)" $1 | awk -F "(" '{print $1}' | awk '{print $2}'`
  do
    t="^$t$"
    if [ -z $cases ]
    then
      cases=$t
    else
      cases=$cases"|"$t
    fi
  done
  go test $2 -run "$cases" $pkg
}

gotestdir() {
  for i in `find ./ -name "*_test.go" | grep third -v`
  do
    echo "Testing "$i" ..."
    gotest $i
    if [ $? -ne 0 ]
    then
      exit $?
    fi
    echo 
  done
}

gotestall() {
  for i in `find $self/src -name "*_test.go" | grep third -v`
  do
    echo "Testing [$i] ..."
    gotest $i
    if [ $? -ne 0 ]
    then
      exit $?
    fi
    echo
  done
}

gobuildall() {
  allpkgs=`find $self/src -type f | xargs grep "^package " | grep third -vw | awk '{print $NF}' | sort | uniq | grep main -v`
  for pkg in $allpkgs
  do
    gobuildpkg $pkg
  done
}

gobuildpkg() {
  if [ -z $1 ]
  then
    pkg=`pwd | awk -F "/" '{print $NF}'`
  else
    pkg=$1
  fi
  path=`find $self/src -name "$pkg"`
  pkg=`echo $path | awk -v "b=$self/src/" '{print substr($1, length(b) + 1)}'`
  echo -n "Build [$pkg] ... "
  go build $pkg
  if [ $? -eq 0 ]
  then
    echo "[Success]"
  else
    echo "[Fail]"
    exit $?
  fi
}

if [ -z $1 ]
then
  build eTunnel
elif [ "x$1" == "xcheck" ]
then
  check
elif [ "x$1" == "xclean" ]
then
  clean
elif [ "x$1" == "xtest" ]
then
  if [ -z $2 ]
  then
    gotestdir
  else
    gotest $2 "-v"
  fi
elif [ "x$1" == "xbuild" ]
then
  gobuildpkg $2
elif [ "x$1" == "xtestall" ]
then
  gotestall
elif [ "x$1" == "xbuildall" ]
then
  gobuildall
else
  echo "unknow parameter [$1]"
  exit -1
fi
