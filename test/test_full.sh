#!/bin/bash
# Written by Claude Sonnet 4.6

FUSE_BINARY="../onionfs"
TEST_DIR="./onionfs_test_env"
LOWER_DIR="$TEST_DIR/lower"
UPPER_DIR="$TEST_DIR/upper"
MOUNT_DIR="$TEST_DIR/mnt"
LOG_FILE="$TEST_DIR/onionfs.log"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

pass() { echo -e "${GREEN}PASSED${NC}"; ((PASS++)); }
fail() { echo -e "${RED}FAILED${NC} — $1"; ((FAIL++)); }

mount_fs() {
    local extra_flags="$1"
    $FUSE_BINARY -u "$UPPER_DIR" -l "$LOWER_DIR" -m "$MOUNT_DIR" $extra_flags > "$LOG_FILE" 2>&1 &
    FUSE_PID=$!
    sleep 1
}

unmount() {
    fusermount -u "$MOUNT_DIR" 2>/dev/null || umount "$MOUNT_DIR" 2>/dev/null
    # wait for the process to actually exit
    wait $FUSE_PID 2>/dev/null
    sleep 0.5
}

setup() {
    unmount
    rm -rf "$TEST_DIR"
    mkdir -p "$LOWER_DIR" "$UPPER_DIR" "$MOUNT_DIR"
}

clean() {
    rm -rf "$TEST_DIR"
}

echo "=============================="
echo "  onionfs Test Suite"
echo "=============================="
echo ""

# -------------------------------------------------------
# TEST 1: Basic layer visibility
# Function:
#  1. Create base.txt in lower dir with content "lower_content"
#  2. Mount OnionFS with lower and upper layers
#  3. Read base.txt from mountpoint
#  4. Verify content matches lower dir content
# -------------------------------------------------------
echo -e "${YELLOW}[Layer Stacking]${NC}"
echo -n "Test 1: File in lower is visible through mountpoint: "
setup
echo "lower_content" > "$LOWER_DIR/base.txt"
mount_fs
if grep -q "lower_content" "$MOUNT_DIR/base.txt" 2>/dev/null; then pass; else fail "base.txt not visible or wrong content"; fi
unmount

# -------------------------------------------------------
# TEST 2: Upper takes precedence over lower
# Function:
#  1. Create shadow.txt in lower dir with content "lower_version"
#  2. Create shadow.txt in upper dir with content "upper_version"
#  3. Mount OnionFS
#  4. Read shadow.txt from mountpoint
#  5. Verify content matches upper dir version
# -------------------------------------------------------
echo -n "Test 2: Upper dir file shadows lower dir file: "
setup
echo "lower_version" > "$LOWER_DIR/shadow.txt"
echo "upper_version" > "$UPPER_DIR/shadow.txt"
mount_fs
if grep -q "upper_version" "$MOUNT_DIR/shadow.txt" 2>/dev/null; then pass; else fail "upper did not shadow lower"; fi
unmount

# -------------------------------------------------------
# TEST 3: Both layers visible in readdir
# Function:
#  1. Create lower_only.txt in lower dir
#  2. Create upper_only.txt in upper dir
#  3. Mount OnionFS
#  4. List mountpoint directory
#  5. Verify both lower and upper files appear in merged listing
# -------------------------------------------------------
echo -n "Test 3: Readdir shows merged view of both layers: "
setup
echo "a" > "$LOWER_DIR/lower_only.txt"
echo "b" > "$UPPER_DIR/upper_only.txt"
mount_fs
ls_out=$(ls "$MOUNT_DIR" 2>/dev/null)
if echo "$ls_out" | grep -q "lower_only.txt" && echo "$ls_out" | grep -q "upper_only.txt"; then pass
else fail "merged readdir missing entries: $ls_out"; fi
unmount

# -------------------------------------------------------
# TEST 4: Copy-on-Write — write to lower file
# Function:
#  1. Create cow.txt in lower dir with content "original"
#  2. Mount OnionFS
#  3. Append "modified" to cow.txt via mountpoint
#  4. Verify file now exists in upper dir with new content
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[Copy-on-Write]${NC}"
echo -n "Test 4: Writing to lower file copies it to upper: "
setup
echo "original" > "$LOWER_DIR/cow.txt"
mount_fs
echo "modified" >> "$MOUNT_DIR/cow.txt" 2>/dev/null
if grep -q "modified" "$UPPER_DIR/cow.txt" 2>/dev/null; then pass; else fail "file not copied to upper on write"; fi
unmount

# -------------------------------------------------------
# TEST 5: CoW — lower file is untouched
# Function:
#  1. Create cow.txt in lower dir with content "original"
#  2. Mount OnionFS
#  3. Append "modified" to cow.txt via mountpoint
#  4. Verify lower dir file still contains only "original" (no "modified")
# -------------------------------------------------------
echo -n "Test 5: Lower file untouched after CoW write: "
setup
echo "original" > "$LOWER_DIR/cow.txt"
mount_fs
echo "modified" >> "$MOUNT_DIR/cow.txt" 2>/dev/null
if grep -q "original" "$LOWER_DIR/cow.txt" && ! grep -q "modified" "$LOWER_DIR/cow.txt"; then pass
else fail "lower file was modified"; fi
unmount

# -------------------------------------------------------
# TEST 6: CoW — read reflects changes
# Function:
#  1. Create cow.txt in lower dir with content "original"
#  2. Mount OnionFS
#  3. Append "modified" to cow.txt via mountpoint
#  4. Read cow.txt from mountpoint
#  5. Verify content includes "modified" (CoW'd copy in upper)
# -------------------------------------------------------
echo -n "Test 6: Mountpoint reflects modified content after CoW: "
setup
echo "original" > "$LOWER_DIR/cow.txt"
mount_fs
echo "modified" >> "$MOUNT_DIR/cow.txt" 2>/dev/null
if grep -q "modified" "$MOUNT_DIR/cow.txt" 2>/dev/null; then pass; else fail "mountpoint does not reflect write"; fi
unmount

# -------------------------------------------------------
# TEST 7: Create new file — goes to upper only
# Function:
#  1. Mount OnionFS (empty upper and lower dirs)
#  2. Create newfile.txt via mountpoint with content "new"
#  3. Verify newfile.txt exists in upper dir
#  4. Confirm new file only appears in upper, not lower
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[File Creation]${NC}"
echo -n "Test 7: New file created via mountpoint appears in upper: "
setup
mount_fs
echo "new" > "$MOUNT_DIR/newfile.txt" 2>/dev/null
if [ -f "$UPPER_DIR/newfile.txt" ]; then pass; else fail "newfile.txt not in upper_dir"; fi
unmount

# -------------------------------------------------------
# TEST 8: New file not in lower
# Function:
#  1. Mount OnionFS (empty upper and lower dirs)
#  2. Create newfile.txt via mountpoint with content "new"
#  3. Check that lower dir does not contain newfile.txt
#  4. Verify writing to mountpoint does not leak into lower
# -------------------------------------------------------
echo -n "Test 8: New file created via mountpoint does not appear in lower: "
setup
mount_fs
echo "new" > "$MOUNT_DIR/newfile.txt" 2>/dev/null
if [ ! -f "$LOWER_DIR/newfile.txt" ]; then pass; else fail "newfile.txt leaked into lower_dir"; fi
unmount

# -------------------------------------------------------
# TEST 9: Whiteout — lower file deleted
# Function:
#  1. Create delete_me.txt in lower dir with content "to_be_deleted"
#  2. Mount OnionFS
#  3. Delete delete_me.txt via mountpoint (rm)
#  4. Verify .wh.delete_me.txt whiteout marker exists in upper dir
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[Whiteout / Deletion]${NC}"
echo -n "Test 9: Deleting a lower file creates a whiteout in upper: "
setup
echo "to_be_deleted" > "$LOWER_DIR/delete_me.txt"
mount_fs
rm "$MOUNT_DIR/delete_me.txt" 2>/dev/null
if [ -f "$UPPER_DIR/.wh.delete_me.txt" ]; then pass; else fail ".wh.delete_me.txt not created in upper"; fi
unmount

# -------------------------------------------------------
# TEST 10: Whiteout — file hidden from mountpoint
# Function:
#  1. Create delete_me.txt in lower dir with content "to_be_deleted"
#  2. Mount OnionFS
#  3. Delete delete_me.txt via mountpoint (rm)
#  4. Verify delete_me.txt is no longer visible from mountpoint
# -------------------------------------------------------
echo -n "Test 10: Whited-out file is hidden from mountpoint: "
setup
echo "to_be_deleted" > "$LOWER_DIR/delete_me.txt"
mount_fs
rm "$MOUNT_DIR/delete_me.txt" 2>/dev/null
if [ ! -f "$MOUNT_DIR/delete_me.txt" ]; then pass; else fail "file still visible after deletion"; fi
unmount

# -------------------------------------------------------
# TEST 11: Whiteout — lower file preserved
# Function:
#  1. Create delete_me.txt in lower dir with content "to_be_deleted"
#  2. Mount OnionFS
#  3. Delete delete_me.txt via mountpoint (rm)
#  4. Verify delete_me.txt still exists in lower dir (not physically deleted)
# -------------------------------------------------------
echo -n "Test 11: Lower file preserved after whiteout: "
setup
echo "to_be_deleted" > "$LOWER_DIR/delete_me.txt"
mount_fs
rm "$MOUNT_DIR/delete_me.txt" 2>/dev/null
if [ -f "$LOWER_DIR/delete_me.txt" ]; then pass; else fail "lower file was physically deleted"; fi
unmount

# -------------------------------------------------------
# TEST 12: Delete upper-only file — no whiteout needed
# Function:
#  1. Create upper_only.txt in upper dir
#  2. Mount OnionFS
#  3. Delete upper_only.txt via mountpoint (rm)
#  4. Verify file removed from upper and no whiteout marker created
# -------------------------------------------------------
echo -n "Test 12: Deleting an upper-only file removes it directly (no whiteout: "
setup
echo "upper_only" > "$UPPER_DIR/upper_only.txt"
mount_fs
rm "$MOUNT_DIR/upper_only.txt" 2>/dev/null
if [ ! -f "$UPPER_DIR/upper_only.txt" ] && [ ! -f "$UPPER_DIR/.wh.upper_only.txt" ]; then pass
else fail "upper-only file deletion left debris"; fi
unmount

# -------------------------------------------------------
# TEST 13: Delete file in both layers — whiteout created
# Function:
#  1. Create both.txt in lower dir with content "lower"
#  2. Create both.txt in upper dir with content "upper"
#  3. Mount OnionFS
#  4. Delete both.txt via mountpoint (rm)
#  5. Verify upper copy removed and .wh.both.txt whiteout created
# -------------------------------------------------------
echo -n "Test 13: Deleting a file in both layers removes upper and creates whiteout: "
setup
echo "lower" > "$LOWER_DIR/both.txt"
echo "upper" > "$UPPER_DIR/both.txt"
mount_fs
rm "$MOUNT_DIR/both.txt" 2>/dev/null
if [ ! -f "$UPPER_DIR/both.txt" ] && [ -f "$UPPER_DIR/.wh.both.txt" ]; then pass
else fail "expected upper deleted + whiteout created"; fi
unmount

# -------------------------------------------------------
# TEST 14: Mkdir — directory created in upper
# Function:
#  1. Mount OnionFS (empty upper and lower dirs)
#  2. Create newdir via mountpoint (mkdir)
#  3. Verify newdir exists in upper dir
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[Mkdir / Rmdir]${NC}"
echo -n "Test 14: mkdir via mountpoint creates directory in upper: "
setup
mount_fs
mkdir "$MOUNT_DIR/newdir" 2>/dev/null
if [ -d "$UPPER_DIR/newdir" ]; then pass; else fail "newdir not created in upper"; fi
unmount

# -------------------------------------------------------
# TEST 15: Mkdir — visible through mountpoint
# Function:
#  1. Mount OnionFS (empty upper and lower dirs)
#  2. Create newdir via mountpoint (mkdir)
#  3. List mountpoint to verify newdir appears
# -------------------------------------------------------
echo -n "Test 15: New directory visible through mountpoint: "
setup
mount_fs
mkdir "$MOUNT_DIR/newdir" 2>/dev/null
if [ -d "$MOUNT_DIR/newdir" ]; then pass; else fail "newdir not visible in mountpoint"; fi
unmount

# -------------------------------------------------------
# TEST 16: Mkdir — lower dir already exists should fail
# Function:
#  1. Create existingdir in lower dir
#  2. Mount OnionFS
#  3. Attempt mkdir existingdir via mountpoint
#  4. Verify operation fails with non-zero exit code (EEXIST)
# -------------------------------------------------------
echo -n "Test 16: mkdir on existing lower directory returns EEXIST: "
setup
mkdir -p "$LOWER_DIR/existingdir"
mount_fs
mkdir "$MOUNT_DIR/existingdir" 2>/dev/null
exit_code=$?
if [ $exit_code -ne 0 ]; then pass; else fail "mkdir did not fail on existing dir (exit code: $exit_code)"; fi
unmount

# -------------------------------------------------------
# TEST 17: Rmdir — lower dir whiteout created
# Function:
#  1. Create lower_dir in lower dir
#  2. Mount OnionFS
#  3. Delete lower_dir via mountpoint (rmdir)
#  4. Verify .wh.lower_dir whiteout marker exists in upper dir
# -------------------------------------------------------
echo -n "Test 17: rmdir on lower directory creates whiteout: "
setup
mkdir -p "$LOWER_DIR/lower_dir"
mount_fs
rmdir "$MOUNT_DIR/lower_dir" 2>/dev/null
if [ -f "$UPPER_DIR/.wh.lower_dir" ]; then pass; else fail ".wh.lower_dir not created"; fi
unmount

# -------------------------------------------------------
# TEST 18: Rmdir — non-empty directory fails
# Function:
#  1. Create nonempty directory in lower dir
#  2. Create file.txt inside nonempty directory
#  3. Mount OnionFS
#  4. Attempt rmdir nonempty via mountpoint
#  5. Verify operation fails with non-zero exit code (ENOTEMPTY)
# -------------------------------------------------------
echo -n "Test 18: rmdir on non-empty directory returns ENOTEMPTY: "
setup
mkdir -p "$LOWER_DIR/nonempty"
echo "file" > "$LOWER_DIR/nonempty/file.txt"
mount_fs
rmdir "$MOUNT_DIR/nonempty" 2>/dev/null
exit_code=$?
if [ $exit_code -ne 0 ]; then pass; else fail "rmdir did not fail on non-empty dir"; fi
unmount

# -------------------------------------------------------
# TEST 19: Rmdir — upper-only dir removed directly
# Function:
#  1. Create upper_only_dir in upper dir
#  2. Mount OnionFS
#  3. Delete upper_only_dir via mountpoint (rmdir)
#  4. Verify dir removed from upper and no whiteout marker created
# -------------------------------------------------------
echo -n "Test 19: rmdir on upper-only empty directory removes it directly: "
setup
mkdir -p "$UPPER_DIR/upper_only_dir"
mount_fs
rmdir "$MOUNT_DIR/upper_only_dir" 2>/dev/null
if [ ! -d "$UPPER_DIR/upper_only_dir" ] && [ ! -f "$UPPER_DIR/.wh.upper_only_dir" ]; then pass
else fail "upper-only dir deletion left debris"; fi
unmount

# -------------------------------------------------------
# TEST 20: Recreate after whiteout clears the whiteout
# Function:
#  1. Create recreate.txt in lower dir with content "original"
#  2. Mount OnionFS
#  3. Delete recreate.txt via mountpoint (rm) to create .wh.recreate.txt
#  4. Create recreate.txt via mountpoint with content "recreated"
#  5. Verify whiteout marker removed and new content visible
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[Whiteout Edge Cases]${NC}"
echo -n "Test 20: Recreating a whited-out file removes the whiteout: "
setup
echo "original" > "$LOWER_DIR/recreate.txt"
mount_fs
rm "$MOUNT_DIR/recreate.txt" 2>/dev/null
echo "recreated" > "$MOUNT_DIR/recreate.txt" 2>/dev/null
if [ ! -f "$UPPER_DIR/.wh.recreate.txt" ] && grep -q "recreated" "$MOUNT_DIR/recreate.txt" 2>/dev/null; then pass
else fail "whiteout not cleared on recreate or content wrong"; fi
unmount

# -------------------------------------------------------
# TEST 21: show-meta flag shows .wh. files
# Function:
#  1. Create delete_me.txt in lower dir
#  2. Mount OnionFS with --show-meta flag
#  3. Delete delete_me.txt via mountpoint (rm) to create .wh.delete_me.txt
#  4. List mountpoint directory
#  5. Verify .wh.delete_me.txt appears in listing (visible due to --show-meta)
# -------------------------------------------------------
echo -n "Test 21: --show-meta flag exposes whiteout files in mountpoint: "
setup
echo "to_be_deleted" > "$LOWER_DIR/delete_me.txt"
mount_fs "--show-meta"
rm "$MOUNT_DIR/delete_me.txt" 2>/dev/null
if ls -la "$MOUNT_DIR" 2>/dev/null | grep -q ".wh.delete_me.txt"; then pass
else fail ".wh.delete_me.txt not visible with --show-meta"; fi
unmount

# -------------------------------------------------------
# TEST 22: without show-meta, whiteout files are hidden
# Function:
#  1. Create delete_me.txt in lower dir
#  2. Mount OnionFS without --show-meta flag
#  3. Delete delete_me.txt via mountpoint (rm) to create .wh.delete_me.txt
#  4. List mountpoint directory
#  5. Verify .wh.delete_me.txt does not appear in listing (hidden by default)
# -------------------------------------------------------
echo -n "Test 22: Without --show-meta, whiteout files are hidden: "
setup
echo "to_be_deleted" > "$LOWER_DIR/delete_me.txt"
mount_fs
rm "$MOUNT_DIR/delete_me.txt" 2>/dev/null
if ls -la "$MOUNT_DIR" 2>/dev/null | grep -q ".wh.delete_me.txt"; then fail ".wh. file visible without --show-meta"
else pass; fi
unmount

# -------------------------------------------------------
# TEST 23: Nested directory visibility
# Function:
#  1. Create nested directory structure in lower: a/b/
#  2. Create deep.txt inside lower/a/b/ with content "deep"
#  3. Mount OnionFS
#  4. Read deep.txt from mountpoint via nested path a/b/deep.txt
#  5. Verify nested file is visible and accessible through merged mount
# -------------------------------------------------------
echo ""
echo -e "${YELLOW}[Nested Directories]${NC}"
echo -n "Test 23: Nested lower files visible through mountpoint: "
setup
mkdir -p "$LOWER_DIR/a/b"
echo "deep" > "$LOWER_DIR/a/b/deep.txt"
mount_fs
if grep -q "deep" "$MOUNT_DIR/a/b/deep.txt" 2>/dev/null; then pass; else fail "nested file not visible"; fi
unmount

# -------------------------------------------------------
# TEST 24: CoW preserves directory structure
# Function:
#  1. Create nested directory structure in lower: nested/dir/
#  2. Create file.txt inside lower/nested/dir/ with content "original"
#  3. Mount OnionFS
#  4. Append "modified" to file.txt via mountpoint nested path
#  5. Verify upper dir creates matching nested/dir structure for CoW'd file
# -------------------------------------------------------
echo -n "Test 24: CoW write to nested lower file creates matching upper dir structure: "
setup
mkdir -p "$LOWER_DIR/nested/dir"
echo "original" > "$LOWER_DIR/nested/dir/file.txt"
mount_fs
echo "modified" >> "$MOUNT_DIR/nested/dir/file.txt" 2>/dev/null
if [ -f "$UPPER_DIR/nested/dir/file.txt" ]; then pass; else fail "upper dir structure not created for CoW"; fi
unmount


# -------------------------------------------------------
# Clean
# -------------------------------------------------------
clean

# -------------------------------------------------------
# Summary
# -------------------------------------------------------
echo ""
echo "=============================="
echo -e "  Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
echo "=============================="

[ $FAIL -eq 0 ] && exit 0 || exit 1