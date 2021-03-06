//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/uber/makisu/lib/storage/metadata"

	"github.com/stretchr/testify/require"
)

// These tests should pass for all FileEntry implementations
func TestFileEntry(t *testing.T) {
	stores := []struct {
		name    string
		fixture func() (bundle *fileEntryTestBundle, cleanup func())
	}{
		{"LocalFileEntry", fileEntryLocalFixture},
	}

	tests := []func(require *require.Assertions, bundle *fileEntryTestBundle){
		testCreate,
		testCreateExisting,
		testCreateFail,
		testMoveFrom,
		testMoveFromExisting,
		testMoveFromWrongState,
		testMoveFromWrongSourcePath,
		testMove,
		testLinkTo,
		testDelete,
		testGetMetadataAndSetMetadata,
		testGetMetadataFail,
		testSetMetadataAt,
		testGetOrSetMetadata,
		testDeleteMetadata,
		testRangeMetadata,
	}

	for _, store := range stores {
		t.Run(store.name, func(t *testing.T) {
			for _, test := range tests {
				testName := runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
				parts := strings.Split(testName, ".")
				t.Run(parts[len(parts)-1], func(t *testing.T) {
					require := require.New(t)
					s, cleanup := store.fixture()
					defer cleanup()
					test(require, s)
				})
			}
		})
	}
}

func testCreate(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1

	fp := fe.GetPath()
	testFileSize := int64(123)

	// Create succeeds with correct state.
	err := fe.Create(s1, testFileSize)
	require.NoError(err)
	info, err := os.Stat(fp)
	require.NoError(err)
	require.Equal(info.Size(), testFileSize)
}

func testCreateExisting(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1

	fp := fe.GetPath()
	testFileSize := int64(123)

	// Create succeeds with correct state.
	err := fe.Create(s1, testFileSize)
	require.NoError(err)
	info, err := os.Stat(fp)
	require.NoError(err)
	require.Equal(info.Size(), testFileSize)

	// Create fails with existing file.
	err = fe.Create(s1, testFileSize-1)
	require.True(os.IsExist(err))
	info, err = os.Stat(fp)
	require.NoError(err)
	require.Equal(info.Size(), testFileSize)
}

func testCreateFail(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s2 := bundle.state2

	fp := fe.GetPath()
	testFileSize := int64(123)

	// Create fails with wrong state.
	err := fe.Create(s2, testFileSize)
	require.Error(err)
	require.True(IsFileStateError(err))
	_, err = os.Stat(fp)
	require.Error(err)
	require.True(os.IsNotExist(err))
}

func testMoveFrom(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1
	s3 := bundle.state3

	fp := fe.GetPath()
	testSourceFile, err := ioutil.TempFile(s3.GetDirectory(), "")
	require.NoError(err)

	// MoveFrom succeeds with correct state and source path.
	err = fe.MoveFrom(s1, testSourceFile.Name())
	require.NoError(err)
	_, err = os.Stat(fp)
	require.NoError(err)
}

func testMoveFromExisting(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1
	s3 := bundle.state3

	fp := fe.GetPath()
	testSourceFile, err := ioutil.TempFile(s3.GetDirectory(), "")
	require.NoError(err)

	// MoveFrom succeeds with correct state and source path.
	err = fe.MoveFrom(s1, testSourceFile.Name())
	require.NoError(err)
	_, err = os.Stat(fp)
	require.NoError(err)

	// MoveFrom fails with existing file.
	testSourceFile2, err := ioutil.TempFile(s3.GetDirectory(), "")
	require.NoError(err)
	err = fe.MoveFrom(s1, testSourceFile2.Name())
	require.True(os.IsExist(err))
	_, err = os.Stat(fp)
	require.NoError(err)
}

func testMoveFromWrongState(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s2 := bundle.state2
	s3 := bundle.state3

	fp := fe.GetPath()
	testSourceFile, err := ioutil.TempFile(s3.GetDirectory(), "")
	require.NoError(err)

	// MoveFrom fails with wrong state.
	err = fe.MoveFrom(s2, testSourceFile.Name())
	require.Error(err)
	require.True(IsFileStateError(err))
	_, err = os.Stat(fp)
	require.Error(err)
	require.True(os.IsNotExist(err))
}

func testMoveFromWrongSourcePath(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1

	fp := fe.GetPath()

	// MoveFrom fails with wrong source path.
	err := fe.MoveFrom(s1, "")
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(fp)
	require.Error(err)
	require.True(os.IsNotExist(err))
}

func testMove(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1
	s2 := bundle.state2
	s3 := bundle.state3

	fn := fe.GetName()
	fp := fe.GetPath()
	testFileSize := int64(123)
	m := getMockMetadataOne()
	m.content = blobFixture(8)
	mm := getMockMetadataMovable()
	mm.content = blobFixture(8)

	// Create file first.
	err := fe.Create(s1, testFileSize)
	require.NoError(err)

	// Write metadata
	updated, err := fe.SetMetadata(m)
	require.NoError(err)
	require.True(updated)
	updated, err = fe.SetMetadata(mm)
	require.NoError(err)
	require.True(updated)

	// Verify metadata is readable.
	mresult := getMockMetadataOne()
	require.NoError(fe.GetMetadata(mresult))
	require.Equal(m.content, mresult.content)

	mmresult := getMockMetadataMovable()
	require.NoError(fe.GetMetadata(mmresult))
	require.Equal(mm.content, mmresult.content)

	// Move file, removes non-movable metadata.
	err = fe.Move(s3)
	require.NoError(err)
	_, err = os.Stat(fp)
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(fe.GetPath())
	require.NoError(err)

	// Verify metadata that's not movable is deleted.
	err = fe.GetMetadata(getMockMetadataOne())
	require.Error(err)
	require.True(os.IsNotExist(err))
	for _, s := range []FileState{s1, s2, s3} {
		_, err = os.Stat(filepath.Join(s.GetDirectory(), fn, getMockMetadataOne().GetSuffix()))
		require.Error(err)
		require.True(os.IsNotExist(err))
	}

	// Verify metadata that's movable should have been moved along with the file entry.
	mmresult = getMockMetadataMovable()
	require.NoError(fe.GetMetadata(mmresult))
	require.Equal(mm.content, mmresult.content)

	_, err = os.Stat(filepath.Join(s3.GetDirectory(), fn))
	require.Nil(err)
	_, err = os.Stat(filepath.Join(s1.GetDirectory(), fn, getMockMetadataMovable().GetSuffix()))
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(s2.GetDirectory(), fn, getMockMetadataMovable().GetSuffix()))
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(s3.GetDirectory(), fn, getMockMetadataMovable().GetSuffix()))
	require.NoError(err)
}

func testLinkTo(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1
	s3 := bundle.state3

	// Create file first.
	testFileSize := int64(123)
	err := fe.Create(s1, testFileSize)
	require.NoError(err)
	testDstFile := filepath.Join(s3.GetDirectory(), "test_dst")

	// LinkTo succeeds with correct source path.
	require.NoError(fe.LinkTo(testDstFile))
	_, err = os.Stat(testDstFile)
	require.NoError(err)

	// LinkTo fails with existing source path.
	require.True(os.IsExist(fe.LinkTo(testDstFile)))
}

func testDelete(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry
	s1 := bundle.state1

	fn := fe.GetName()
	fp := fe.GetPath()
	testFileSize := int64(123)
	m := getMockMetadataOne()
	m.content = blobFixture(8)
	mm := getMockMetadataMovable()
	mm.content = blobFixture(8)

	// Create file first.
	err := fe.Create(s1, testFileSize)
	require.NoError(err)

	// Write metadata.
	updated, err := fe.SetMetadata(m)
	require.NoError(err)
	require.True(updated)
	updated, err = fe.SetMetadata(mm)
	require.NoError(err)
	require.True(updated)

	// Delete.
	err = fe.Delete()
	require.NoError(err)

	// Verify the data file and metadata files are all deleted.
	_, err = os.Stat(fp)
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(s1.GetDirectory(), fn, getMockMetadataOne().GetSuffix()))
	require.Error(err)
	require.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(s1.GetDirectory(), fn, getMockMetadataMovable().GetSuffix()))
	require.Error(err)
	require.True(os.IsNotExist(err))
}

func testGetMetadataAndSetMetadata(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	m := getMockMetadataOne()
	m.content = blobFixture(8)

	// Write metadata.
	updated, err := fe.SetMetadata(m)
	require.NoError(err)
	require.True(updated)

	updated, err = fe.SetMetadata(m)
	require.NoError(err)
	require.False(updated)

	// Read metadata.
	result := getMockMetadataOne()
	require.NoError(fe.GetMetadata(result))
	require.Equal(m.content, result.content)

	// Set metadata shorter.
	m.content = blobFixture(4)
	updated, err = fe.SetMetadata(m)
	require.NoError(err)
	require.True(updated)

	// Read metadata.
	result = getMockMetadataOne()
	require.NoError(fe.GetMetadata(result))
	require.Equal(m.content, result.content)
}

func testGetMetadataFail(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	m1 := getMockMetadataOne()
	m2 := getMockMetadataTwo()

	// Invalid read.
	err := fe.GetMetadata(m1)
	require.True(os.IsNotExist(err))

	// Invalid read.
	err = fe.GetMetadata(m2)
	require.True(os.IsNotExist(err))
}

func testSetMetadataAt(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	m := getMockMetadataOne()
	m.content = []byte{1, 2, 3, 4}

	updated, err := fe.SetMetadata(m)
	require.NoError(err)
	require.True(updated)

	updated, err = fe.SetMetadataAt(m, []byte{5, 5}, 1)
	require.NoError(err)
	require.True(updated)

	updated, err = fe.SetMetadataAt(m, []byte{5, 5}, 1)
	require.NoError(err)
	require.False(updated)

	result := getMockMetadataOne()
	require.NoError(fe.GetMetadata(result))
	require.Equal([]byte{1, 5, 5, 4}, result.content)
}

func testGetOrSetMetadata(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	original := []byte("foo")

	m := getMockMetadataOne()
	m.content = original

	// First GetOrSet should write.
	require.NoError(fe.GetOrSetMetadata(m))
	require.Equal(original, m.content)

	m.content = []byte("bar")

	// Second GetOrSet should read.
	require.NoError(fe.GetOrSetMetadata(m))
	require.Equal(original, m.content)
}

func testDeleteMetadata(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	m := getMockMetadataOne()
	m.content = blobFixture(8)

	_, err := fe.SetMetadata(m)
	require.NoError(err)

	require.NoError(fe.GetMetadata(getMockMetadataOne()))

	require.NoError(fe.DeleteMetadata(m))

	err = fe.GetMetadata(getMockMetadataOne())
	require.Error(err)
	require.True(os.IsNotExist(err))
}

func testRangeMetadata(require *require.Assertions, bundle *fileEntryTestBundle) {
	fe := bundle.entry

	ms := []metadata.Metadata{
		getMockMetadataOne(),
		getMockMetadataTwo(),
		getMockMetadataMovable(),
	}
	for _, m := range ms {
		_, err := fe.SetMetadata(m)
		require.NoError(err)
	}

	var result []metadata.Metadata
	require.NoError(fe.RangeMetadata(func(md metadata.Metadata) error {
		result = append(result, md)
		return nil
	}))

	require.ElementsMatch(ms, result)
}
