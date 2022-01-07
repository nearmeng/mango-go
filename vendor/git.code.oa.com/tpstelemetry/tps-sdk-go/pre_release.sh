set -e

help()
{
   printf "\n"
   printf "Usage: $0 -t tag\n"
   printf "\t-t Unreleased tag. Update all go.mod with this tag.\n"
   exit 1 # Exit script after printing help
}

while getopts "t:" opt
do
   case "$opt" in
      t ) TAG="$OPTARG" ;;
      ? ) help ;; # Print help
   esac
done

if [ -z "$TAG" ]
then
   printf "Tag is missing\n";
   help
fi

# Validate semver
SEMVER_REGEX="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
if [[ "${TAG}" =~ ${SEMVER_REGEX} ]]; then
	printf "${TAG} is valid semver tag.\n"
else
	printf "${TAG} is not a valid semver tag.\n"
	exit 1
fi

TAG_FOUND=`git tag --list ${TAG}`
if [[ ${TAG_FOUND} = ${TAG} ]] ; then
        printf "Tag ${TAG} already exists\n"
        exit 1
fi

if ! git diff --quiet; then \
	printf "Working tree is not clean, can't proceed with the release process\n"
	exit 1
fi

git checkout -b pre_release_${TAG} master
PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | sed 's/^\.\///' | sort)

for dir in $PACKAGE_DIRS; do
	cp "${dir}/go.mod" "${dir}/go.mod.bak"
	sed "s/git.code.oa.com\/tpstelemetry\/tps-sdk-go\([^ ]*\) v[0-9]*\.[0-9]*\.[0-9]*/git.code.oa.com\/tpstelemetry\/tps-sdk-go\1 ${TAG}/" "${dir}/go.mod.bak" >"${dir}/go.mod"
	rm -f "${dir}/go.mod.bak"
done

# sed version metric
VERSION_FILE="version.go"
cp ${VERSION_FILE} ${VERSION_FILE}.bak
sed "s/const version = \"[0-9.]*\"/const version = \"${TAG:1}\"/g" ${VERSION_FILE}.bak >${VERSION_FILE}
rm -f ${VERSION_FILE}.bak

# make precommit to check code
make precommit

git add .
git commit -m "Prepare for releasing ${TAG}"

printf "Now run following to verify the changes.\ngit diff master\n"
printf "\nPlease update CHANGELOG.md\n"
printf "\nThen push the changes to upstream\n"
