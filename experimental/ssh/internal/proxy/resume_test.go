package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendBufferReplaysWhatThePeerMissed(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("hello ")))
	require.NoError(t, b.append([]byte("world")))

	// The peer wrote only the first 6 bytes before the connection broke.
	missing, err := b.replayFrom(6)
	require.NoError(t, err)
	assert.Equal(t, "world", string(missing))
}

func TestSendBufferReplayFromCurrentOffsetIsEmpty(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("hello")))

	missing, err := b.replayFrom(5)
	require.NoError(t, err)
	assert.Empty(t, missing)
}

func TestSendBufferAckDiscardsThePrefix(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("aaaabbbb")))
	b.ack(4)

	// Everything from the acknowledged offset is still replayable.
	missing, err := b.replayFrom(4)
	require.NoError(t, err)
	assert.Equal(t, "bbbb", string(missing))

	// The discarded prefix is not.
	_, err = b.replayFrom(3)
	assert.ErrorIs(t, err, errReplayUnavailable)
}

func TestSendBufferAckFreesTheWindow(t *testing.T) {
	b := newSendBuffer(8)
	require.NoError(t, b.append([]byte("12345678")))
	require.ErrorIs(t, b.append([]byte("9")), errSendWindowExhausted)

	b.ack(8)
	assert.NoError(t, b.append([]byte("9")))
}

func TestSendBufferIgnoresStaleAndImpossibleAcks(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abcdef")))
	b.ack(4)

	// A stale ack must not rewind the buffer, and an ack beyond what we sent must not
	// discard bytes the peer cannot have received.
	b.ack(2)
	b.ack(99)

	missing, err := b.replayFrom(4)
	require.NoError(t, err)
	assert.Equal(t, "ef", string(missing))
}

func TestSendBufferRejectsAnImpossibleReplayOffset(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abc")))

	_, err := b.replayFrom(4)
	require.Error(t, err)
	assert.NotErrorIs(t, err, errReplayUnavailable)
}

func TestSendBufferReplayIsACopy(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abcdef")))

	missing, err := b.replayFrom(0)
	require.NoError(t, err)
	missing[0] = 'z'

	again, err := b.replayFrom(0)
	require.NoError(t, err)
	assert.Equal(t, "abcdef", string(again), "mutating a replay must not corrupt the buffer")
}

// TestSendBufferFillReturnsError verifies that when the replay buffer fills,
// append returns an error rather than panicking or silently dropping data.
func TestSendBufferFillReturnsError(t *testing.T) {
	b := newSendBuffer(100)

	// Fill to the limit: 100 bytes
	require.NoError(t, b.append([]byte("a")))
	require.NoError(t, b.append(make([]byte, 99)))

	// Next append should fail with errSendWindowExhausted
	err := b.append([]byte("b"))
	require.ErrorIs(t, err, errSendWindowExhausted)

	// But the buffer should still be accessible for replay
	missing, err := b.replayFrom(0)
	require.NoError(t, err)
	assert.Len(t, missing, 100)
}
