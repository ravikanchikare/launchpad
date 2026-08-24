//! Native macOS alert panels for update feedback (legacy HarnezPad parity).

const builtin = @import("builtin");

const ObjcObject = *anyopaque;
const Selector = *anyopaque;
const ObjcClass = *anyopaque;

extern fn objc_getClass(name: [*:0]const u8) ?ObjcClass;
extern fn sel_registerName(name: [*:0]const u8) ?Selector;
extern fn objc_msgSend() callconv(.c) void;

const SendObject0 = *const fn (?ObjcObject, Selector) callconv(.c) ?ObjcObject;
const SendObject1 = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject2 = *const fn (?ObjcObject, Selector, ?ObjcObject, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendInteger0 = *const fn (?ObjcObject, Selector) callconv(.c) isize;
const SendVoid1Object = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) void;
const SendVoid1Integer = *const fn (?ObjcObject, Selector, isize) callconv(.c) void;

pub fn presentInformational(title: []const u8, message: []const u8) void {
    if (comptime builtin.is_test or builtin.os.tag != .macos) return;
    presentInformationalImpl(title, message) catch {};
}

pub fn presentConfirm(title: []const u8, message: []const u8) bool {
    if (comptime builtin.is_test or builtin.os.tag != .macos) return false;
    return presentConfirmImpl(title, message) catch false;
}

fn presentInformationalImpl(title: []const u8, message: []const u8) !void {
    const alert = try makeAlert(title, message);
    _ = addButton(alert, "OK");
    _ = sendInteger0(alert, try selector("runModal"));
}

fn presentConfirmImpl(title: []const u8, message: []const u8) !bool {
    const alert = try makeAlert(title, message);
    _ = addButton(alert, "Install and Restart");
    _ = addButton(alert, "Cancel");
    const response = sendInteger0(alert, try selector("runModal"));
    return response == 1000; // NSAlertFirstButtonReturn
}

fn makeAlert(title: []const u8, message: []const u8) !ObjcObject {
    const alert_class = objc_getClass("NSAlert") orelse return error.AppKitClassUnavailable;
    const alert = sendObject0(alert_class, try selector("alloc")) orelse
        return error.AppKitObjectCreationFailed;
    const initialized = sendObject0(alert, try selector("init")) orelse
        return error.AppKitObjectCreationFailed;
    sendVoid1Object(initialized, try selector("setMessageText:"), try nsString(title));
    sendVoid1Object(initialized, try selector("setInformativeText:"), try nsString(message));
    sendVoid1Integer(initialized, try selector("setAlertStyle:"), 1); // NSAlertStyleInformational
    return initialized;
}

fn addButton(alert: ObjcObject, label: []const u8) ?ObjcObject {
    return sendObject1(alert, selector("addButtonWithTitle:") catch return null, nsString(label) catch return null);
}

fn nsString(value: []const u8) !ObjcObject {
    const ns_string = objc_getClass("NSString") orelse return error.AppKitClassUnavailable;
    var buffer: [512]u8 = undefined;
    if (value.len >= buffer.len) return error.AppKitStringTooLong;
    @memcpy(buffer[0..value.len], value);
    buffer[value.len] = 0;
    return sendObject1(ns_string, try selector("stringWithUTF8String:"), @ptrCast(&buffer)) orelse
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

fn sendInteger0(receiver: ?ObjcObject, action: Selector) isize {
    const send: SendInteger0 = @ptrCast(&objc_msgSend);
    return send(receiver, action);
}

fn sendVoid1Object(receiver: ?ObjcObject, action: Selector, value: ?ObjcObject) void {
    const send: SendVoid1Object = @ptrCast(&objc_msgSend);
    send(receiver, action, value);
}

fn sendVoid1Integer(receiver: ?ObjcObject, action: Selector, value: isize) void {
    const send: SendVoid1Integer = @ptrCast(&objc_msgSend);
    send(receiver, action, value);
}
