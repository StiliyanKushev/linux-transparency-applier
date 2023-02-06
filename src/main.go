package main

//#cgo LDFLAGS: -lX11 -lXext
//#include <stdio.h>
//#include <stdlib.h>
//#include <X11/Xlib.h>
//#include <X11/Xutil.h>
//#include <X11/Xos.h>
//#include <X11/Xatom.h>
//#include <X11/extensions/shape.h>
//unsigned long XGETPIXEL(XImage *ximage, int x, int y) {
//	return XGetPixel(ximage, x, y);
//}
import "C"
import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

//export CreateSandboxParentWindow
func CreateSandboxParentWindow(x, y, width, height int) int {
	display := C.XOpenDisplay(nil)

	if display == nil {
		panic("Failed to open display.")
	}

	screen := C.XDefaultScreen(display)

	// CreateSimpleWindow is InputOutput by default
	window := C.XCreateSimpleWindow(
		// display (root display)
		display,
		// parent window (root window)
		C.XRootWindow(display, screen),
		// x, y, width, height, border_width
		C.int(x), C.int(y), C.uint(width), C.uint(height), 1,
		// border color
		C.XBlackPixel(display, screen),
		// background color
		C.XBlackPixel(display, screen))

	// var attrs C.XSetWindowAttributes
	// attrs.override_redirect = C.True
	// C.XChangeWindowAttributes(display, window, C.CWOverrideRedirect, &attrs)

	C.XMapWindow(display, window)
	C.XFlush(display)

	return int(window)
}

//export BindXNestedToWindow
func BindXNestedToWindow(windowId int) int {
	displayId := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1000) + 10
	cmd := exec.Command(`Xnest`, fmt.Sprintf(":%d", displayId), `-parent`, fmt.Sprint(windowId))

	if err := cmd.Start(); err != nil {
		fmt.Println(cmd.String())
		log.Fatal(err)
		return 0
	}

	return displayId
}

//export GetWindowIdsByDisplayId
func GetWindowIdsByDisplayId(displayId int, sleepTime int) *C.char {
	// sometimes we want to sleep to make sure the program is ready
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	// define our resizable slice array
	windowIds := []int{}

	// convert display id to c_string
	C_displayId := C.CString(fmt.Sprintf(":%d", displayId))
	defer C.free(unsafe.Pointer(C_displayId))

	// open the specific display, get the root window
	dpy := C.XOpenDisplay(C_displayId)
	defer C.XCloseDisplay(dpy)
	rootWin := C.XDefaultRootWindow(dpy)

	// define pointers for all return values
	var root, parent C.Window
	var children *C.Window
	var numChildren C.uint

	// search the root window of our display for a window with the provided windowName
	if C.XQueryTree(dpy, rootWin, &root, &parent, &children, &numChildren) != 0 {
		for i := 0; i < int(numChildren); i++ {
			var child = *(*C.Window)(unsafe.Pointer(uintptr(unsafe.Pointer(children)) +
				uintptr(i)*unsafe.Sizeof(*children)))

			found_windowId := int(child)
			windowIds = append(windowIds, found_windowId)
		}
		C.XFree(unsafe.Pointer(children))
	}

	// we convert our int array to a string
	// and then that go string to a c_string
	// as it's easier to pass to nodejs than an array of integers
	// through the ffi-napi interface
	strSlice := make([]string, len(windowIds))
	for i, v := range windowIds {
		strSlice[i] = fmt.Sprintf("%d", v)
	}

	return C.CString(strings.Join(strSlice, `,`))
}

//export TransformWindow
func TransformWindow(windowId, displayId int) {
	// convert display id to c_string
	C_displayId := C.CString(fmt.Sprintf(":%d", displayId))
	defer C.free(unsafe.Pointer(C_displayId))

	// open the specific display
	dpy := C.XOpenDisplay(C_displayId)

	// get the root window and enable event masks
	rootWin := C.XDefaultRootWindow(dpy)

	var attr C.XWindowAttributes
	C.XGetWindowAttributes(dpy, rootWin, &attr)
	current_width := int(attr.width)
	current_height := int(attr.height)

	// xmove this window to 0,0 and resize to max width and height (ak root window size)
	C.XMoveResizeWindow(dpy, C.ulong(windowId), 0, 0, C.uint(current_width), C.uint(current_height))
	C.XFlush(dpy)
}

//export ResetWindowShape
func ResetWindowShape(windowId, displayId int) {
	// convert display id to c_string
	C_displayId := C.CString(fmt.Sprintf(":%d", displayId))
	defer C.free(unsafe.Pointer(C_displayId))

	// open the specific display
	dpy := C.XOpenDisplay(C_displayId)

	// reset the window shape back to it's original
	// before we potentially remove stuff
	// that's to not leave empty spots on the window
	C.XShapeCombineMask(dpy, C.ulong(windowId), C.ShapeBounding, 0, 0, C.None, C.ShapeSet)
	C.XSync(dpy, 0)
	C.XFlush(dpy)
}

//export CreateExcluderShape
func CreateExcluderShape(C_hexColor *C.char, windowId, displayId int) {
	// parse the hex color to rgb values
	hexColor := C.GoString(C_hexColor)

	// convert hexColor string to int (skipping '#')
	i, _ := strconv.ParseInt(hexColor[1:], 16, 32)

	// extract red, green and blue value using bitwise operation
	filterRed := int((i >> 16) & 0xff)
	filterGreen := int((i >> 8) & 0xff)
	filterBlue := int(i & 0xff)

	// convert display id to c_string
	C_displayId := C.CString(fmt.Sprintf(":%d", displayId))
	defer C.free(unsafe.Pointer(C_displayId))

	// open the specific display
	dpy := C.XOpenDisplay(C_displayId)

	// calcuate the width and height of the window
	var attr C.XWindowAttributes
	C.XGetWindowAttributes(dpy, C.ulong(windowId), &attr)
	current_width := int(attr.width)
	current_height := int(attr.height)

	// init a region we're going to gradually expand
	region := C.XCreateRegion()

	// get image of the current state of the window
	image := C.XGetImage(dpy, C.ulong(windowId), 0, 0, C.uint(current_width), C.uint(current_height), C.AllPlanes, C.ZPixmap)

	var pixelCount = 0

	// loop each pixel of the image data
	for x := 0; x < current_width; x++ {
		for y := 0; y < current_height; y++ {
			pixel := C.XGETPIXEL(image, C.int(x), C.int(y))
			red := int((pixel >> 16) & 0xff)
			green := int((pixel >> 8) & 0xff)
			blue := int(pixel & 0xff)

			// check if the color is the one we're targeting
			if red == filterRed && green == filterGreen && blue == filterBlue {
				// there are fewer pixels to account for this way
				pixelCount++
				rect := C.XRectangle{C.short(x), C.short(y), 1, 1}
				C.XUnionRectWithRegion(&rect, region, region)
			}
		}
	}

	// wait for all async requests to be ready
	C.XSync(dpy, 0)
	C.XFlush(dpy)

	// apply the mask
	C.XShapeCombineRegion(dpy, C.ulong(windowId), C.ShapeBounding, 0, 0, region, C.ShapeSubtract)

	// wait for the mask to be ready
	C.XSync(dpy, 0)
	C.XFlush(dpy)

	fmt.Println(`pixelCount`, pixelCount)
}

//export LinkEventsWithChild
func LinkEventsWithChild(parentWindowId, displayId int, C_stringChildWindowIds, C_hexColor *C.char) {
	// open the root display where the parent window lives
	rootDpy := C.XOpenDisplay(nil)

	// enable resize event listening for the parent window
	C.XSelectInput(rootDpy, C.ulong(parentWindowId), C.StructureNotifyMask|C.ExposureMask)

	// open the nested display where the children live
	C_displayId := C.CString(fmt.Sprintf(":%d", displayId))
	defer C.free(unsafe.Pointer(C_displayId))
	childDpy := C.XOpenDisplay(C_displayId)

	// convert the window ids of the children to []int
	stringChildWindowIds := C.GoString(C_stringChildWindowIds)
	strSlice := strings.Split(stringChildWindowIds, ",")
	var childWindowIds []int
	for _, s := range strSlice {
		i, _ := strconv.Atoi(s)
		childWindowIds = append(childWindowIds, i)
	}

	//  #define KeyPress		   2
	//  #define KeyRelease		   3
	//  #define ButtonPress		   4
	//  #define ButtonRelease	   5
	//  #define MotionNotify	   6
	//  #define EnterNotify		   7
	//  #define LeaveNotify		   8
	//  #define FocusIn			   9
	//  #define FocusOut		   10
	//  #define KeymapNotify	   11
	//  #define Expose			   12
	//  #define GraphicsExpose	   13
	//  #define NoExpose		   14
	//  #define VisibilityNotify   15
	//  #define CreateNotify	   16
	//  #define DestroyNotify	   17
	//  #define UnmapNotify		   18
	//  #define MapNotify		   19
	//  #define MapRequest		   20
	//  #define ReparentNotify	   21
	//  #define ConfigureNotify	   22
	//  #define ConfigureRequest   23
	//  #define GravityNotify	   24
	//  #define ResizeRequest	   25
	//  #define CirculateNotify	   26
	//  #define CirculateRequest   27
	//  #define PropertyNotify	   28
	//  #define SelectionClear	   29
	//  #define SelectionRequest   30
	//  #define SelectionNotify	   31
	//  #define ColormapNotify	   32
	//  #define ClientMessage	   33
	//  #define MappingNotify	   34
	//  #define GenericEvent	   35
	//  #define LASTEvent		   36

	// handle events from parent to child
	// : open the actual root display
	// : listen for resize events of the parent window
	// : on resize event of parent window resize child window id

	go func() {
		for {
			// we constantly listen for pops from the event Q
			var event C.XEvent
			C.XNextEvent(rootDpy, &event)

			// we are only interested in the event type of the struct
			eventData := (*[1]uintptr)(unsafe.Pointer(&event))
			eventType := eventData[0]

			// calcuate the new width and height of the window
			var attr C.XWindowAttributes
			C.XGetWindowAttributes(rootDpy, C.ulong(parentWindowId), &attr)
			current_width := int(attr.width)
			current_height := int(attr.height)

			fmt.Println(`received event`, eventType)

			// the window configuration changed here
			if eventType == C.ConfigureNotify {
				fmt.Println(`resize/move window`)
				// xmove this window to 0,0 and resize to max width and height (aka root window size)
				// do that for every child window specified
				for _, childWindowId := range childWindowIds {
					C.XMoveResizeWindow(childDpy, C.ulong(childWindowId), 0, 0, C.uint(current_width), C.uint(current_height))
					C.XFlush(childDpy)
				}

				// we also apply the transparency mask to the parent window
				CreateExcluderShape(C_hexColor, parentWindowId, 0)
			}

			// the window received an expose event from the window manager
			if eventType == C.Expose {
				fmt.Println(`refresh window`)
				C.XFillRectangle(
					rootDpy,
					C.ulong(parentWindowId),
					C.XDefaultGC(rootDpy, 0), 0, 0, C.uint(current_width), C.uint(current_height))
				C.XFlush(rootDpy)
			}
		}
	}()

	// handle events from child to parent
	// : open the nested display from the provided display id
	// : listen for close, minimize, maximize events of the child window id
	// : on such events act on the parent window id instead

	// todo:
}

func main() {
	// shared library
}
