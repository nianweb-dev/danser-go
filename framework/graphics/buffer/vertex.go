package buffer

import (
	"fmt"
	"github.com/wieku/danser-go/framework/graphics/history"
	"github.com/wieku/danser-go/framework/graphics/shader"
	"github.com/wieku/danser-go/framework/statistic"
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/pkg/errors"
)

// VertexSlice points to a portion of (or possibly whole) vertex array. It is used as a pointer,
// contrary to Go's builtin slices. This is, so that append can be 'in-place'. That's for the good,
// because Begin/End-ing a VertexSlice would become super confusing, if append returned a new
// VertexSlice.
//
// It also implements all basic slice-like operations: appending, sub-slicing, etc.
//
// Note that you need to Begin a VertexSlice before getting or updating it's elements or drawing it.
// After you're done with it, you need to End it.
type VertexSlice struct {
	va   *vertexArray
	i, j int
}

// MakeVertexSlice allocates a new vertex array with specified capacity and returns a VertexSlice
// that points to it's first len elements.
//
// Note, that a vertex array is specialized for a specific shader and can't be used with another
// shader.
func MakeVertexSlice(shader *shader.Shader, len, cap int) *VertexSlice {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice{
		va: newVertexArray(shader, cap),
		i:  0,
		j:  len,
	}
}

// VertexFormat returns the format of vertex attributes inside the underlying vertex array of this
// VertexSlice.
func (vs *VertexSlice) VertexFormat() shader.AttrFormat {
	return vs.va.format
}

// Stride returns the number of float32 elements occupied by one vertex.
func (vs *VertexSlice) Stride() int {
	return vs.va.stride / 4
}

// Len returns the length of the VertexSlice (number of vertices).
func (vs *VertexSlice) Len() int {
	return vs.j - vs.i
}

// Cap returns the capacity of an underlying vertex array.
func (vs *VertexSlice) Cap() int {
	return vs.va.cap - vs.i
}

// SetLen resizes the VertexSlice to length len.
func (vs *VertexSlice) SetLen(len int) {
	vs.End() // vs must have been Begin-ed before calling this method
	*vs = vs.grow(len)
	vs.Begin()
}

// grow returns supplied vs with length changed to len. Allocates new underlying vertex array if
// necessary. The original content is preserved.
func (vs VertexSlice) grow(len int) VertexSlice {
	if len <= vs.Cap() {
		// capacity sufficient
		return VertexSlice{
			va: vs.va,
			i:  vs.i,
			j:  vs.i + len,
		}
	}

	// grow the capacity
	newCap := vs.Cap()
	if newCap < 1024 {
		newCap += newCap
	} else {
		newCap += newCap / 4
	}
	if newCap < len {
		newCap = len
	}
	newVs := VertexSlice{
		va: newVertexArray(vs.va.shader, newCap),
		i:  0,
		j:  len,
	}
	// preserve the original content
	newVs.Begin()
	newVs.Slice(0, vs.Len()).SetVertexData(vs.VertexData())
	newVs.End()
	return newVs
}

// Slice returns a sub-slice of this VertexSlice covering the range [i, j) (relative to this
// VertexSlice).
//
// Note, that the returned VertexSlice shares an underlying vertex array with the original
// VertexSlice. Modifying the contents of one modifies corresponding contents of the other.
func (vs *VertexSlice) Slice(i, j int) *VertexSlice {
	if i < 0 || j < i || j > vs.va.cap {
		panic("failed to slice vertex slice: index out of range")
	}
	return &VertexSlice{
		va: vs.va,
		i:  vs.i + i,
		j:  vs.i + j,
	}
}

// SetVertexData sets the contents of the VertexSlice.
//
// The data is a slice of float32's, where each vertex attribute occupies a certain number of
// elements. Namely, Float occupies 1, Vec2 occupies 2, Vec3 occupies 3 and Vec4 occupies 4. The
// attribues in the data slice must be in the same order as in the vertex format of this Vertex
// Slice.
//
// If the length of vertices does not match the length of the VertexSlice, this methdo panics.
func (vs *VertexSlice) SetVertexData(data []float32) {
	if len(data)/vs.Stride() != vs.Len() {
		fmt.Println(len(data)/vs.Stride(), vs.Len())
		panic("set vertex data: wrong length of vertices")
	}
	vs.va.setVertexData(vs.i, vs.j, data)
}

// VertexData returns the contents of the VertexSlice.
//
// The data is in the same format as with SetVertexData.
func (vs *VertexSlice) VertexData() []float32 {
	return vs.va.vertexData(vs.i, vs.j)
}

// Draw draws the content of the VertexSlice.
func (vs *VertexSlice) Draw() {
	vs.va.draw(vs.i, vs.j)
}

// Begin binds the underlying vertex array. Calling this method is necessary before using the VertexSlice.
func (vs *VertexSlice) Begin() {
	vs.va.begin()
}

// BeginDraw binds vertex array without binding underlying vbo. Use this if you only want to draw elements.
func (vs *VertexSlice) BeginDraw() {
	vs.va.beginDraw()
}

// End unbinds the underlying vertex array. Call this method when you're done with VertexSlice.
func (vs *VertexSlice) End() {
	vs.va.end()
}

// End unbinds the underlying vertex array. Call this method when you're done with VertexSlice draw.
func (vs *VertexSlice) EndDraw() {
	vs.va.endDraw()
}

// Delete removes this vertex array from GPU memory.
func (vs *VertexSlice) Delete() {
	vs.va.delete()
}

type vertexArray struct {
	vao, vbo uint32
	cap      int
	format   shader.AttrFormat
	stride   int
	offset   []int
	shader   *shader.Shader
}

const vertexArrayMinCap = 4

func newVertexArray(shdr *shader.Shader, cap int) *vertexArray {
	if cap < vertexArrayMinCap {
		cap = vertexArrayMinCap
	}

	va := &vertexArray{
		cap:    cap,
		format: shdr.VertexFormat(),
		stride: shdr.VertexFormat().Size(),
		offset: make([]int, len(shdr.VertexFormat())),
		shader: shdr,
	}

	offset := 0
	for i, attr := range va.format {
		switch attr.Type {
		case shader.Float, shader.Vec2, shader.Vec3, shader.Vec4:
		default:
			panic(errors.New("failed to create vertex array: invalid attribute type"))
		}
		va.offset[i] = offset
		offset += attr.Type.Size()
	}

	gl.GenVertexArrays(1, &va.vao)

	va.bindVAO()

	gl.GenBuffers(1, &va.vbo)

	va.bindVBO()

	emptyData := make([]byte, cap*va.stride)
	gl.BufferData(gl.ARRAY_BUFFER, len(emptyData), gl.Ptr(emptyData), gl.DYNAMIC_DRAW)

	for i, attr := range va.format {
		loc := gl.GetAttribLocation(shdr.Handle, gl.Str(attr.Name+"\x00"))

		var size int32
		switch attr.Type {
		case shader.Float:
			size = 1
		case shader.Vec2:
			size = 2
		case shader.Vec3:
			size = 3
		case shader.Vec4:
			size = 4
		}

		gl.VertexAttribPointer(
			uint32(loc),
			size,
			gl.FLOAT,
			false,
			int32(va.stride),
			gl.PtrOffset(va.offset[i]),
		)
		gl.EnableVertexAttribArray(uint32(loc))
	}

	va.unbindVBO()
	va.unbindVAO()

	runtime.SetFinalizer(va, (*vertexArray).delete)

	return va
}

func (va *vertexArray) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteVertexArrays(1, &va.vao)
		gl.DeleteBuffers(1, &va.vbo)
	})
}

func (va *vertexArray) begin() {
	va.bindVAO()
	va.bindVBO()
}

func (va *vertexArray) beginDraw() {
	va.bindVAO()
}

func (va *vertexArray) end() {
	va.unbindVBO()
	va.unbindVAO()
}

func (va *vertexArray) endDraw() {
	va.unbindVAO()
}

func (va *vertexArray) bindVAO() {
	history.Push(gl.VERTEX_ARRAY_BINDING)
	statistic.Increment(statistic.VAOBinds)
	gl.BindVertexArray(va.vao)
}

func (va *vertexArray) unbindVAO() {
	handle := history.Pop(gl.VERTEX_ARRAY_BINDING)
	if handle != 0 {
		statistic.Increment(statistic.VAOBinds)
	}
	gl.BindVertexArray(handle)
}

func (va *vertexArray) bindVBO() {
	history.Push(gl.ARRAY_BUFFER_BINDING)
	statistic.Increment(statistic.VBOBinds)
	gl.BindBuffer(gl.ARRAY_BUFFER, va.vbo)
}

func (va *vertexArray) unbindVBO() {
	handle := history.Pop(gl.ARRAY_BUFFER_BINDING)
	if handle != 0 {
		statistic.Increment(statistic.VBOBinds)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, handle)
}

func (va *vertexArray) draw(i, j int) {
	statistic.Add(statistic.VerticesDrawn, int64(j-i))
	statistic.Increment(statistic.DrawCalls)
	gl.DrawArrays(gl.TRIANGLES, int32(i), int32(j-i))
}

func (va *vertexArray) setVertexData(i, j int, data []float32) {
	if j-i == 0 {
		// avoid setting 0 bytes of buffer data
		return
	}
	statistic.Add(statistic.VertexUpload, int64(len(data)/(va.stride/4)))
	gl.BufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
}

func (va *vertexArray) vertexData(i, j int) []float32 {
	if j-i == 0 {
		// avoid getting 0 bytes of buffer data
		return nil
	}
	data := make([]float32, (j-i)*va.stride/4)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
	return data
}
