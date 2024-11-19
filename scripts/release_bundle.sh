#!/bin/bash

if [ $# -eq 0 ]
then
    echo "Error: VERSION argument required"
    echo "Usage: $0 VERSION"
    exit 1
fi

VERSION=$1
OUTPUT_BINARY="mir"
TEMP_FOLDER=.release
README=book/src/running_mir/binary.md
LICENSE=LICENSE

rm -rf $TEMP_FOLDER

mkdir -p $TEMP_FOLDER/linux-amd64
mkdir -p $TEMP_FOLDER/linux-arm64
mkdir -p $TEMP_FOLDER/windows-amd64
mkdir -p $TEMP_FOLDER/windows-arm64
echo "${VERSION}" > $TEMP_FOLDER/linux-amd64/VERSION
echo "${VERSION}" > $TEMP_FOLDER/linux-arm64/VERSION
echo "${VERSION}" > $TEMP_FOLDER/windows-amd64/VERSION
echo "${VERSION}" > $TEMP_FOLDER/windows-arm64/VERSION
cat $LICENSE > $TEMP_FOLDER/linux-amd64/LICENSE
cat $LICENSE > $TEMP_FOLDER/linux-arm64/LICENSE
cat $LICENSE > $TEMP_FOLDER/windows-amd64/LICENSE
cat $LICENSE > $TEMP_FOLDER/windows-arm64/LICENSE
cat $README > $TEMP_FOLDER/linux-amd64/README.md
cat $README > $TEMP_FOLDER/linux-arm64/README.md
cat $README > $TEMP_FOLDER/windows-amd64/README.md
cat $README > $TEMP_FOLDER/windows-arm64/README.md

# Linux amd64
echo "Building Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o "./$TEMP_FOLDER/linux-amd64/${OUTPUT_BINARY}" cmds/mir/main.go

# Linux arm64
echo "Building Linux arm64..."
GOOS=linux GOARCH=arm64 go build -o "./$TEMP_FOLDER/linux-arm64/${OUTPUT_BINARY}" cmds/mir/main.go

# Windows amd64
echo "Building Windows amd64..."
GOOS=windows GOARCH=amd64 go build -o "./$TEMP_FOLDER/windows-amd64/${OUTPUT_BINARY}" cmds/mir/main.go

# Windows arm64
echo "Building Windows arm64..."
GOOS=windows GOARCH=arm64 go build -o "./$TEMP_FOLDER/windows-arm64/${OUTPUT_BINARY}" cmds/mir/main.go

# Create tar.gz for Linux amd64
echo "Creating tar.gz for Linux amd64..."
tar -czf $TEMP_FOLDER/linux-amd64.tar.gz -C $TEMP_FOLDER/linux-amd64 .

# Create tar.gz for Linux arm64
echo "Creating tar.gz for Linux arm64..."
tar -czf $TEMP_FOLDER/linux-arm64.tar.gz -C $TEMP_FOLDER/linux-arm64 .

# Create zip for Windows amd64
echo "Creating zip for Windows amd64..."
(cd $TEMP_FOLDER/windows-amd64 && zip -r ../windows-amd64.zip .)

# Create zip for Windows arm64
echo "Creating zip for Windows arm64..."
(cd $TEMP_FOLDER/windows-arm64 && zip -r ../windows-arm64.zip .)

rm -rf $TEMP_FOLDER/linux-amd64
rm -rf $TEMP_FOLDER/linux-arm64
rm -rf $TEMP_FOLDER/windows-amd64
rm -rf $TEMP_FOLDER/windows-arm64

echo ""
echo "Mir $VERSION bundle released 🚀!"
