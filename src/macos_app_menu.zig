//! Inserts legacy HarnezPad application-menu items (Settings, Check for Updates)
//! and routes Quit through the app's command handler instead of a raw terminate.

const std = @import("std");
const builtin = @import("builtin");

const ObjcObject = *anyopaque;
const Selector = *anyopaque;
const ObjcClass = *anyopaque;

pub const CommandFn = *const fn (command: [*:0]const u8) callconv(.c) void;

extern fn objc_getClass(name: [*:0]const u8) ?ObjcClass;
extern fn objc_allocateClassPair(superclass: ?ObjcClass, name: [*:0]const u8, extra_bytes: usize) ?ObjcClass;
extern fn objc_registerClassPair(cls: ObjcClass) void;
extern fn class_addMethod(cls: ObjcClass, name: Selector, implementation: *const anyopaque, types: [*:0]const u8) bool;
extern fn sel_registerName(name: [*:0]const u8) ?Selector;
extern fn objc_msgSend() callconv(.c) void;

const SendObject0 = *const fn (?ObjcObject, Selector) callconv(.c) ?ObjcObject;
const SendObject1 = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject2 = *const fn (?ObjcObject, Selector, ?ObjcObject, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject3 = *const fn (?ObjcObject, Selector, ?ObjcObject, Selector, ?ObjcObject) callconv(.c) ?ObjcObject;
const SendObject1Integer = *const fn (?ObjcObject, Selector, usize) callconv(.c) ?ObjcObject;
const SendInteger0 = *const fn (?ObjcObject, Selector) callconv(.c) usize;
const SendVoid1Object = *const fn (?ObjcObject, Selector, ?ObjcObject) callconv(.c) void;
const SendVoid1Integer = *const fn (?ObjcObject, Selector, isize) callconv(.c) void;
const SendVoid2ObjectInteger = *const fn (?ObjcObject, Selector, ?ObjcObject, usize) callconv(.c) void;

var command_handler: ?CommandFn = null;
var menu_target: ?ObjcObject = null;

pub fn install(handler: CommandFn) void {
    if (comptime builtin.os.tag != .macos) return;
    command_handler = handler;
    patchApplicationMenu() catch {};
}

fn patchApplicationMenu() !void {
    const application_class = objc_getClass("NSApplication") orelse return error.AppKitClassUnavailable;
    const application = sendObject0(application_class, try selector("sharedApplication")) orelse
        return error.AppKitObjectCreationFailed;
    const main_menu = sendObject0(application, try selector("mainMenu")) orelse
        return error.AppKitMenuUnavailable;
    if (sendInteger0(main_menu, try selector("numberOfItems")) == 0) return error.AppKitMenuUnavailable;

    const app_menu_item = sendObject1Integer(main_menu, try selector("itemAtIndex:"), 0) orelse
        return error.AppKitMenuUnavailable;
    const app_menu = sendObject0(app_menu_item, try selector("submenu")) orelse
        return error.AppKitMenuUnavailable;

    const target = try menuTarget();
    try insertLegacyItems(app_menu, target);
    try replaceQuitItem(app_menu, target);
}

fn insertLegacyItems(app_menu: ObjcObject, target: ObjcObject) !void {
    const separator_class = objc_getClass("NSMenuItem") orelse return error.AppKitClassUnavailable;
    const separator = sendObject0(separator_class, try selector("separatorItem")) orelse
        return error.AppKitObjectCreationFailed;

    try insertItem(app_menu, 2, try commandItem(
        target,
        "Settings…",
        "harnezpad.settings",
        ",",
        0x0100, // NSEventModifierFlagCommand
    ));
    try insertItem(app_menu, 3, try commandItem(
        target,
        "Check for Updates…",
        "harnezpad.check-updates",
        "",
        0,
    ));
    insertItem(app_menu, 4, separator) catch {};
}

fn replaceQuitItem(app_menu: ObjcObject, target: ObjcObject) !void {
    const count = sendInteger0(app_menu, try selector("numberOfItems"));
    var index: usize = count;
    while (index > 0) {
        index -= 1;
        const item = sendObject1Integer(app_menu, try selector("itemAtIndex:"), index) orelse continue;
        const title = sendObject0(item, try selector("title")) orelse continue;
        const title_c = sendObject0(title, try selector("UTF8String")) orelse continue;
        const title_bytes: [*:0]const u8 = @ptrCast(title_c);
        if (!std.mem.startsWith(u8, std.mem.span(title_bytes), "Quit")) continue;

        sendVoid1Object(item, try selector("setTarget:"), target);
        sendVoid1Object(item, try selector("setAction:"), try selector("harnezpadMenuCommand:"));
        sendVoid1Object(item, try selector("setRepresentedObject:"), try nsString("harnezpad.quit"));
        return;
    }
}

fn commandItem(
    target: ObjcObject,
    title: []const u8,
    command: []const u8,
    key: []const u8,
    modifiers: u32,
) !ObjcObject {
    const item_class = objc_getClass("NSMenuItem") orelse return error.AppKitClassUnavailable;
    const allocated = sendObject0(item_class, try selector("alloc")) orelse
        return error.AppKitObjectCreationFailed;
    const item = sendObject3(
        allocated,
        try selector("initWithTitle:action:keyEquivalent:"),
        try nsString(title),
        try selector("harnezpadMenuCommand:"),
        try nsString(key),
    ) orelse return error.AppKitObjectCreationFailed;
    sendVoid1Object(item, try selector("setTarget:"), target);
    sendVoid1Object(item, try selector("setRepresentedObject:"), try nsString(command));
    sendVoid1Integer(item, try selector("setKeyEquivalentModifierMask:"), @intCast(modifiers));
    return item;
}

fn insertItem(menu: ObjcObject, index: usize, item: ObjcObject) !void {
    sendVoid2ObjectInteger(menu, try selector("insertItem:atIndex:"), item, index);
}

fn menuTarget() !ObjcObject {
    if (menu_target) |existing| return existing;
    const target_class = try ensureMenuTargetClass();
    menu_target = sendObject0(target_class, try selector("new")) orelse return error.AppKitObjectCreationFailed;
    return menu_target.?;
}

fn ensureMenuTargetClass() !ObjcClass {
    if (objc_getClass("HarnezPadAppMenuTarget")) |existing| return existing;
    const superclass = objc_getClass("NSObject") orelse return error.AppKitClassUnavailable;
    const cls = objc_allocateClassPair(superclass, "HarnezPadAppMenuTarget", 0) orelse
        return error.AppKitClassCreationFailed;
    if (!class_addMethod(
        cls,
        try selector("harnezpadMenuCommand:"),
        @ptrCast(&menuCommandMethod),
        "v@:@",
    )) return error.AppKitMethodCreationFailed;
    objc_registerClassPair(cls);
    return cls;
}

fn menuCommandMethod(_: ?ObjcObject, _: ?Selector, sender: ?ObjcObject) callconv(.c) void {
    const handler = command_handler orelse return;
    const item = sender orelse return;
    const represented = sendObject0(item, selector("representedObject") catch return) orelse return;
    const command_c = sendObject0(represented, selector("UTF8String") catch return) orelse return;
    handler(@ptrCast(command_c));
}

fn nsString(value: []const u8) !ObjcObject {
    const ns_string = objc_getClass("NSString") orelse return error.AppKitClassUnavailable;
    var buffer: [128]u8 = undefined;
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

fn sendObject3(receiver: ?ObjcObject, action: Selector, first: ?ObjcObject, second: Selector, third: ?ObjcObject) ?ObjcObject {
    const send: SendObject3 = @ptrCast(&objc_msgSend);
    return send(receiver, action, first, second, third);
}

fn sendObject1Integer(receiver: ?ObjcObject, action: Selector, value: usize) ?ObjcObject {
    const send: SendObject1Integer = @ptrCast(&objc_msgSend);
    return send(receiver, action, value);
}

fn sendInteger0(receiver: ?ObjcObject, action: Selector) usize {
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

fn sendVoid2ObjectInteger(receiver: ?ObjcObject, action: Selector, first: ?ObjcObject, second: usize) void {
    const send: SendVoid2ObjectInteger = @ptrCast(&objc_msgSend);
    send(receiver, action, first, second);
}
