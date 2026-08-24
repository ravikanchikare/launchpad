//! AppKit chrome that fills the two gaps in Native SDK 0.7's shell views:
//! SF Symbol-backed icon buttons and pointer-backed window drag regions.
//!
//! The sidebar button remains an SDK-owned `.icon_button`, preserving command,
//! focus, accessibility, and automation semantics. We locate that NSButton by
//! the identifier the SDK assigns from `ShellView.label` and apply an SF
//! Symbol. Only the transparent drag views are app-owned adopted surfaces.

const native_sdk = @import("native_sdk");

const geometry = native_sdk.geometry;

const sidebar_drag_label = "titlebar-drag-sidebar";
const content_drag_label = "titlebar-drag-content";

const main_webview_label = "main";

pub const ActionFn = *const fn (context: *anyopaque) callconv(.c) void;

const ObjcObject = *anyopaque;
const Selector = *anyopaque;
const ObjcClass = *anyopaque;

extern fn objc_getClass(name: [*:0]const u8) ?ObjcClass;
extern fn objc_allocateClassPair(superclass: ?ObjcClass, name: [*:0]const u8, extra_bytes: usize) ?ObjcClass;
extern fn objc_registerClassPair(cls: ObjcClass) void;
extern fn class_addMethod(cls: ObjcClass, name: Selector, implementation: *const anyopaque, types: [*:0]const u8) bool;
extern fn sel_registerName(name: [*:0]const u8) ?Selector;
extern fn objc_msgSend() callconv(.c) void;

const SendObject0 = *const fn (?ObjcObject, Selector) callconv(.c) ?ObjcObject;
const SendObject1 = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject2 = *const fn (?ObjcObject, Selector, ?ObjcObject, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject1Integer = *const fn (?ObjcObject, Selector, usize) callconv(.c) ?ObjcObject;
const SendInteger0 = *const fn (?ObjcObject, Selector) callconv(.c) usize;
const SendBool1Object = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) bool;
const SendVoid1Object = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) void;
const SendVoid1Bool = *const fn (?ObjcObject, Selector, bool) callconv(.c) void;
const SendVoid1Integer = *const fn (?ObjcObject, Selector, isize) callconv(.c) void;

var action_context: ?*anyopaque = null;
var drag_action: ?ActionFn = null;

pub fn install(
    runtime: *native_sdk.Runtime,
    context: *anyopaque,
    drag_callback: ActionFn,
) !void {
    action_context = context;
    drag_action = drag_callback;

    clearViewBackground("sidebar-accessory");
    customizeSidebarButton();
    const sidebar_drag = try makeDragRegion();
    const content_drag = try makeDragRegion();

    try runtime.adoptViewSurface(1, sidebar_drag_label, sidebar_drag);
    try runtime.adoptViewSurface(1, content_drag_label, content_drag);
    // Surface adoption transfers first responder to each adopted view. Restore
    // it to the primary WebView after chrome installation.
    runtime.focusView(1, main_webview_label) catch {};
}

pub fn setDragRegions(
    runtime: *native_sdk.Runtime,
    sidebar: geometry.RectF,
    content: geometry.RectF,
) !void {
    try setDragRegion(runtime, sidebar_drag_label, sidebar);
    try setDragRegion(runtime, content_drag_label, content);
}

fn setDragRegion(runtime: *native_sdk.Runtime, label: []const u8, frame: geometry.RectF) !void {
    const visible = frame.width > 0 and frame.height > 0;
    _ = try runtime.updateView(1, label, .{ .visible = visible });
    if (visible) {
        _ = try runtime.updateView(1, label, .{ .frame = frame });
    }
}

fn clearViewBackground(name: [*:0]const u8) void {
    const view = findViewNamed(name) catch return;
    const color_class = objc_getClass("NSColor") orelse return;
    const clear = sendObject0(color_class, selector("clearColor") catch return) orelse return;
    const clear_cg = sendObject0(clear, selector("CGColor") catch return) orelse return;
    sendVoid1Bool(view, selector("setWantsLayer:") catch return, true);
    const layer = sendObject0(view, selector("layer") catch return) orelse return;
    sendVoid1Object(layer, selector("setBackgroundColor:") catch return, clear_cg);
}

fn customizeSidebarButton() void {
    const button = findViewNamed("sidebar-toggle") catch return;
    const ns_button = objc_getClass("NSButton") orelse return;
    if (!sendBool1Object(button, selector("isKindOfClass:") catch return, ns_button)) return;
    const image_class = objc_getClass("NSImage") orelse return;
    const symbol_name = nsString("sidebar.left") catch return;
    const accessibility = nsString("Toggle sidebar") catch return;
    const image = sendObject2(
        image_class,
        selector("imageWithSystemSymbolName:accessibilityDescription:") catch return,
        symbol_name,
        accessibility,
    ) orelse return; // Keep the declared Unicode title on older macOS.

    sendVoid1Object(button, selector("setTitle:") catch return, nsString("") catch return);
    sendVoid1Object(button, selector("setImage:") catch return, image);
    sendVoid1Integer(button, selector("setImagePosition:") catch return, 1); // NSImageOnly
    sendVoid1Bool(button, selector("setBordered:") catch return, false);
    sendVoid1Object(button, selector("setToolTip:") catch return, accessibility);
}

fn makeDragRegion() !ObjcObject {
    const drag_class = try ensureDragViewClass();
    return sendObject0(drag_class, try selector("new")) orelse error.AppKitObjectCreationFailed;
}

fn ensureDragViewClass() !ObjcClass {
    if (objc_getClass("HarnezPadTitlebarDragView")) |existing| return existing;
    const superclass = objc_getClass("NSView") orelse return error.AppKitClassUnavailable;
    const cls = objc_allocateClassPair(superclass, "HarnezPadTitlebarDragView", 0) orelse
        return error.AppKitClassCreationFailed;
    if (!class_addMethod(
        cls,
        try selector("mouseDown:"),
        @ptrCast(&dragMouseDownMethod),
        "v@:@",
    )) return error.AppKitMethodCreationFailed;
    objc_registerClassPair(cls);
    return cls;
}

fn dragMouseDownMethod(_: ?ObjcObject, _: ?Selector, _: ?ObjcObject) callconv(.c) void {
    const callback = drag_action orelse return;
    const context = action_context orelse return;
    callback(context);
}

fn findViewNamed(name: [*:0]const u8) !ObjcObject {
    const windows = try applicationWindows();
    var index = sendInteger0(windows, try selector("count"));
    const identifier = try nsString(name);
    while (index > 0) {
        index -= 1;
        const window = sendObject1Integer(windows, try selector("objectAtIndex:"), index) orelse continue;
        const content = sendObject0(window, try selector("contentView")) orelse continue;
        if (findDescendant(content, identifier)) |view| return view;
    }
    return error.AppKitViewUnavailable;
}

fn applicationWindows() !ObjcObject {
    const application_class = objc_getClass("NSApplication") orelse return error.AppKitClassUnavailable;
    const application = sendObject0(application_class, try selector("sharedApplication")) orelse
        return error.AppKitObjectCreationFailed;
    return sendObject0(application, try selector("windows")) orelse error.AppKitViewUnavailable;
}

fn findDescendant(view: ObjcObject, identifier: ObjcObject) ?ObjcObject {
    const own_identifier = sendObject0(view, selector("identifier") catch return null);
    if (own_identifier != null and sendBool1Object(
        own_identifier,
        selector("isEqualToString:") catch return null,
        identifier,
    )) return view;

    const subviews = sendObject0(view, selector("subviews") catch return null) orelse return null;
    var index = sendInteger0(subviews, selector("count") catch return null);
    // Native SDK orders higher shell layers later. Search backward so the
    // titlebar accessory is found before descending into WKWebView internals.
    while (index > 0) {
        index -= 1;
        const child = sendObject1Integer(subviews, selector("objectAtIndex:") catch return null, index) orelse continue;
        const child_identifier = sendObject0(child, selector("identifier") catch return null);
        if (child_identifier != null and sendBool1Object(
            child_identifier,
            selector("isEqualToString:") catch return null,
            identifier,
        )) return child;
    }
    index = sendInteger0(subviews, selector("count") catch return null);
    while (index > 0) {
        index -= 1;
        const child = sendObject1Integer(subviews, selector("objectAtIndex:") catch return null, index) orelse continue;
        if (findDescendant(child, identifier)) |match| return match;
    }
    return null;
}

fn nsString(value: [*:0]const u8) !ObjcObject {
    const ns_string = objc_getClass("NSString") orelse return error.AppKitClassUnavailable;
    return sendObject1(ns_string, try selector("stringWithUTF8String:"), @ptrCast(@constCast(value))) orelse
        error.AppKitObjectCreationFailed;
}

fn selector(name: [*:0]const u8) !Selector {
    return sel_registerName(name) orelse error.AppKitSelectorUnavailable;
}

fn sendObject0(receiver: ?ObjcObject, action: Selector) ?ObjcObject {
    const send: SendObject0 = @ptrCast(&objc_msgSend);
    return send(receiver, action);
}

fn sendObject1(receiver: ?ObjcObject, action: Selector, first: ?ObjcObject) ?ObjcObject {
    const send: SendObject1 = @ptrCast(&objc_msgSend);
    return send(receiver, action, first);
}

fn sendObject2(receiver: ?ObjcObject, action: Selector, first: ?ObjcObject, second: ?ObjcObject) ?ObjcObject {
    const send: SendObject2 = @ptrCast(&objc_msgSend);
    return send(receiver, action, first, second);
}

fn sendObject1Integer(receiver: ?ObjcObject, action: Selector, value: usize) ?ObjcObject {
    const send: SendObject1Integer = @ptrCast(&objc_msgSend);
    return send(receiver, action, value);
}

fn sendInteger0(receiver: ?ObjcObject, action: Selector) usize {
    const send: SendInteger0 = @ptrCast(&objc_msgSend);
    return send(receiver, action);
}

fn sendBool1Object(receiver: ?ObjcObject, action: Selector, value: ?ObjcObject) bool {
    const send: SendBool1Object = @ptrCast(&objc_msgSend);
    return send(receiver, action, value);
}

fn sendVoid1Object(receiver: ?ObjcObject, action: Selector, value: ?ObjcObject) void {
    const send: SendVoid1Object = @ptrCast(&objc_msgSend);
    send(receiver, action, value);
}

fn sendVoid1Bool(receiver: ?ObjcObject, action: Selector, value: bool) void {
    const send: SendVoid1Bool = @ptrCast(&objc_msgSend);
    send(receiver, action, value);
}

fn sendVoid1Integer(receiver: ?ObjcObject, action: Selector, value: isize) void {
    const send: SendVoid1Integer = @ptrCast(&objc_msgSend);
    send(receiver, action, value);
}
