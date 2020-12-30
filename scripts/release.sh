#!/bin/bash

# make sure we have passed a tag as version
if [ "$1" ]; then
  tag="$1"
else
  echo "correct usage ./release.sh <tag>"
  exit 1
fi

# create signed tag
git tag -s "v$tag" -m "Release $tag"

# build the docker images
./scripts/build.sh "$tag" # create version tag
./scripts/build.sh        # create latest tag

echo ""
echo ""
echo "Release $tag is ready!"
echo "Please validate the changes, and once everything is confirmed, run the following:"
echo ""
echo "Update the git repository:"
echo ""
echo "$ git push && git push --tags"
echo ""
echo "Update the docker images:"
echo ""
echo "$ ./scripts/push.sh $tag"
echo "$ ./scripts/push.sh"
echo ""
echo ""
echo "Happy hacking ;D"
