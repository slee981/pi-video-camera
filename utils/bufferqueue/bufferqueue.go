/*
** BUFFERQUEUE is a simple linked list that keeps 
** track of the first and last nodes, but importantly 
** also has a "maxLength" attribute that will specify 
** conditions of when to update the head. Additionally, 
** this implements a lock condition for updating the 
** queue so as to not add or delete an element while another 
** resource is using it. 
**
** An example use case is to keep a streaming queue 
** of a certain maximum size - so as new elements are added 
** to the queue, the oldest elements are removed.
*/

package bufferqueue

import (
	"fmt"
	"sync"

	"gocv.io/x/gocv"
)

type Node struct {
	data gocv.Mat
	next *Node
}

type BufferQueue struct {
	length     int
	maxLength  int
	isWritable bool

	head *Node
	tail *Node

	mu *sync.Mutex
}

func NewBufferQueue(n int) *BufferQueue {
	return &BufferQueue{
		length: 0,
		maxLength: n,
		isWritable: true,
		mu: new(sync.Mutex),
	}
}

func NewNode(d gocv.Mat, n *Node) *Node {
	return &Node{
		data: d,
		next: n,
	}
}

// we only care about pushing new data into the queue
// cleanup is handled automatically anytime a new 
// node is added that exceeds the "maxLength" of the buffer
func (bq *BufferQueue) Push(d gocv.Mat) (bool, error) {
	data := gocv.NewMat()
	d.CopyTo(&data)

	// lock queue 
	bq.Lock()
	defer bq.Unlock()

	// create new node 
	n := NewNode(data, nil)

	// get current last node 
	l := bq.Last()
	if l == nil {
		// no last node, also no first node
		// init the list 
		bq.SetFirst(n)
	} else {
		l.SetNext(n)
	}

    bq.SetLast(n)
    return true, nil
}

/********************************************************************
** HELPERS TO ACCESS ATTRIBUTES 
*/

// bufferqueue helpers

// external

func (bq *BufferQueue) SetFirst(n *Node) (bool, error) {
    // only called when first node is added 
    bq.head = n
    return true, nil
}

func (bq *BufferQueue) SetLast(n *Node) (bool, error) {

    // set new node to end of list 
	bq.addTail(n)

	// manage length
    // if length is less than maxLength, increment 
    // otherwise, update the head to head.next
    bq.manageLength()
    return true, nil
}

func (bq *BufferQueue) First() *Node {
    return bq.head
}

func (bq *BufferQueue) Last() *Node {
    return bq.tail
}

func (bq *BufferQueue) Length() int {
    return bq.length
}

func (bq *BufferQueue) MaxLength() int {
    return bq.maxLength
}

func (bq *BufferQueue) Lock() {
	bq.mu.Lock()
}

func (bq *BufferQueue) Unlock() {
	bq.mu.Unlock()
}

func (bq *BufferQueue) IsWritable() bool {
	return bq.isWritable
}

func (bq *BufferQueue) LockWrite() {
	bq.isWritable = false
}

func (bq *BufferQueue) UnlockWrite() {
	bq.isWritable = true
}

func (bq *BufferQueue) ToString() {
	for n := bq.head; n != nil; n = n.next {
		fmt.Print(n.data, ", ")
	}
	fmt.Println()
}

// internal 

func (bq *BufferQueue) addTail(n *Node) {
    bq.tail = n
}

func (bq *BufferQueue) manageLength() {
    // check if new length is longer than maxLength
	// if yes, remove first node and reset to next node 
	if bq.Length() >= bq.MaxLength() {
		bq.SetFirst(bq.First().Next())
	} else {
		bq.incrementLength()
	}
}

func (bq *BufferQueue) incrementLength() {
    bq.length++
}


// node helpers

// external 

func (n *Node) SetNext(nn *Node) (bool, error) {
    n.next = nn
    return true, nil
}

func (n *Node) Next() *Node {
    return n.next
}

func (n *Node) GetData() gocv.Mat {
	return n.data
}

