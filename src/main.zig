//! HarnezPad Native SDK hybrid host.
//!
//! Native SDK owns the macOS window, titlebar control, menus, shortcuts,
//! status item, lifecycle, and the authenticated Go helper. The existing
//! React application remains the primary surface through a trusted WebView.

const std = @import("std");
const builtin = @import("builtin");
const runner = @import("runner");
const native_sdk = @import("native_sdk");
const macos_chrome = @import("macos_chrome.zig");
const macos_app_menu = @import("macos_app_menu.zig");
const macos_alert = @import("macos_alert.zig");

pub const panic = std.debug.FullPanic(native_sdk.debug.capturePanic);

const geometry = native_sdk.geometry;

pub const bundle_id = "com.harnezai.launchpad";
pub const window_label = "main";
pub const main_webview_label = "main";
pub const window_width: f32 = 1030;
pub const window_height: f32 = 760;

pub const bootstrap_command = "harnezpad.bootstrap";
pub const sidebar_command = "harnezpad.sidebar.toggle";
pub const quit_bridge_command = "harnezpad.app.quit";
pub const update_alert_command = "harnezpad.update.alert";
pub const update_confirm_command = "harnezpad.update.confirm";
pub const update_complete_command = "harnezpad.update.installComplete";
pub const titlebar_drag_command = "harnezpad.titlebar.setDragRegions";

const open_command = "harnezpad.open";
const launch_command = "harnezpad.launch";
const keys_command = "harnezpad.keys";
const settings_command = "harnezpad.settings";
const check_updates_command = "harnezpad.check-updates";
const quit_command = "harnezpad.quit";
const native_chrome_timer_id: u64 = 0x4152_4355; // "ARCU"

const bridge_origins = [_][]const u8{
    "zero://app",
    "http://127.0.0.1:5173",
};
const allowed_origins = [_][]const u8{
    "zero://app",
    "http://127.0.0.1:5173",
};
const app_permissions = [_][]const u8{
    native_sdk.security.permission_view,
    native_sdk.security.permission_command,
    native_sdk.security.permission_network,
    native_sdk.security.permission_clipboard,
    native_sdk.security.permission_window,
};
const command_permission = [_][]const u8{native_sdk.security.permission_command};
const clipboard_permission = [_][]const u8{native_sdk.security.permission_clipboard};
const window_permission = [_][]const u8{native_sdk.security.permission_window};

const bridge_policies = [_]native_sdk.BridgeCommandPolicy{
    .{ .name = bootstrap_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = sidebar_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = quit_bridge_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = update_alert_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = update_confirm_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = update_complete_command, .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = titlebar_drag_command, .permissions = &window_permission, .origins = &bridge_origins },
};
const builtin_policies = [_]native_sdk.BridgeCommandPolicy{
    .{ .name = "native-sdk.command.invoke", .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.command.list", .permissions = &command_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.clipboard.writeText", .permissions = &clipboard_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.webview.create", .permissions = &window_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.webview.list", .permissions = &window_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.webview.setFrame", .permissions = &window_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.webview.navigate", .permissions = &window_permission, .origins = &bridge_origins },
    .{ .name = "native-sdk.webview.close", .permissions = &window_permission, .origins = &bridge_origins },
};

const tray_items = [_]native_sdk.TrayMenuItem{
    .{ .id = 1, .label = "Open HarnezPad", .command = open_command },
    .{ .separator = true },
    .{ .id = 2, .label = "Settings…", .command = settings_command },
    .{ .id = 3, .label = "Check for Updates…", .command = check_updates_command },
    .{ .separator = true },
    .{ .id = 4, .label = "Quit HarnezPad", .command = quit_command },
};

const shell_views = [_]native_sdk.ShellView{
    .{
        .label = main_webview_label,
        .kind = .webview,
        .url = "zero://app",
        .fill = true,
        .layer = 10,
        .role = "HarnezPad application",
        .accessibility_label = "HarnezPad",
    },
    .{
        .label = "sidebar-accessory",
        .kind = .titlebar_accessory,
        .x = 84,
        .y = 7,
        .width = 38,
        .height = 36,
        .layer = 30,
        .role = "Sidebar controls",
    },
    .{
        .label = "sidebar-toggle",
        .kind = .icon_button,
        .parent = "sidebar-accessory",
        .x = 3,
        .y = 3,
        .width = 30,
        .height = 30,
        .layer = 31,
        .text = "◧",
        .command = sidebar_command,
        .accessibility_label = "Toggle sidebar",
    },
    .{
        .label = "titlebar-drag-sidebar",
        .kind = .stack,
        .x = 122,
        .y = 0,
        .width = 114,
        .height = 52,
        .layer = 29,
    },
    .{
        .label = "titlebar-drag-content",
        .kind = .stack,
        .x = 0,
        .y = 0,
        .width = 1,
        .height = 52,
        .layer = 29,
        .visible = false,
    },
};

const shell_windows = [_]native_sdk.ShellWindow{.{
    .label = window_label,
    .title = "HarnezPad",
    .width = window_width,
    .height = window_height,
    .min_width = 840,
    .min_height = 620,
    .restore_state = true,
    .restore_policy = .clamp_to_visible_screen,
    .titlebar = .hidden_inset_tall,
    .close_policy = .hide,
    .views = &shell_views,
}};

pub const shell_scene: native_sdk.ShellConfig = .{ .windows = &shell_windows };

const HelperState = enum {
    starting,
    ready,
    failed,
    stopped,
};

const ReadyEvent = struct {
    @"type": []const u8,
    url: []const u8,
    token: []const u8,
};

const SpinMutex = struct {
    inner: std.atomic.Mutex = .unlocked,

    fn lock(self: *SpinMutex) void {
        while (!self.inner.tryLock()) std.atomic.spinLoopHint();
    }

    fn unlock(self: *SpinMutex) void {
        self.inner.unlock();
    }
};

pub const HarnezPadApp = struct {
    env_map: *std.process.Environ.Map,
    io: std.Io,
    helper_enabled: bool = true,
    native_chrome_enabled: bool = true,
    native_chrome_installed: bool = false,

    runtime: ?*native_sdk.Runtime = null,
    appearance: native_sdk.ColorScheme = .light,
    sidebar_open: bool = true,

    helper_mutex: SpinMutex = .{},
    helper_state: HelperState = .starting,
    helper_url_storage: [256]u8 = [_]u8{0} ** 256,
    helper_url_len: usize = 0,
    helper_token_storage: [256]u8 = [_]u8{0} ** 256,
    helper_token_len: usize = 0,
    helper_error_storage: [256]u8 = [_]u8{0} ** 256,
    helper_error_len: usize = 0,
    helper_path_storage: [std.fs.max_path_bytes]u8 = [_]u8{0} ** std.fs.max_path_bytes,
    helper_path_len: usize = 0,
    parent_pid_storage: [32]u8 = [_]u8{0} ** 32,
    parent_pid_len: usize = 0,
    status_icon_storage: [std.fs.max_path_bytes]u8 = [_]u8{0} ** std.fs.max_path_bytes,
    status_icon_len: usize = 0,
    helper_pid: std.atomic.Value(i32) = .init(0),
    stopping: std.atomic.Value(bool) = .init(false),
    helper_thread: ?std.Thread = null,

    bridge_handlers: [bridge_policies.len]native_sdk.BridgeHandler = undefined,

    pub fn app(self: *HarnezPadApp) native_sdk.App {
        return .{
            .context = self,
            .name = "HarnezPad",
            .source = native_sdk.frontend.productionSource(.{
                .dist = "frontend/dist",
                .entry = "index.html",
                .origin = "zero://app",
                .spa_fallback = true,
            }),
            .source_fn = source,
            .scene_fn = scene,
            .start_fn = start,
            .event_fn = event,
            .stop_fn = stop,
        };
    }

    fn source(context: *anyopaque) anyerror!native_sdk.WebViewSource {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        return native_sdk.frontend.sourceFromEnv(self.env_map, .{
            .dist = "frontend/dist",
            .entry = "index.html",
            .origin = "zero://app",
            .spa_fallback = true,
        });
    }

    fn scene(context: *anyopaque) anyerror!native_sdk.ShellConfig {
        _ = context;
        return shell_scene;
    }

    pub fn bridge(self: *HarnezPadApp) native_sdk.BridgeDispatcher {
        self.bridge_handlers = .{
            .{ .name = bootstrap_command, .context = self, .invoke_fn = bridgeBootstrap },
            .{ .name = sidebar_command, .context = self, .invoke_fn = bridgeToggleSidebar },
            .{ .name = quit_bridge_command, .context = self, .invoke_fn = bridgeQuit },
            .{ .name = update_alert_command, .context = self, .invoke_fn = bridgeUpdateAlert },
            .{ .name = update_confirm_command, .context = self, .invoke_fn = bridgeUpdateConfirm },
            .{ .name = update_complete_command, .context = self, .invoke_fn = bridgeQuit },
            .{ .name = titlebar_drag_command, .context = self, .invoke_fn = bridgeSetTitlebarDragRegions },
        };
        return .{
            .policy = .{
                .enabled = true,
                .permissions = &app_permissions,
                .commands = &bridge_policies,
            },
            .registry = .{ .handlers = &self.bridge_handlers },
        };
    }

    pub fn configureLaunch(self: *HarnezPadApp, executable_path: []const u8, pid: i32) bool {
        const executable_dir = std.fs.path.dirname(executable_path) orelse return false;
        if (!self.resolveHelperPath(executable_path, executable_dir)) return false;
        const parent_pid = std.fmt.bufPrint(&self.parent_pid_storage, "{d}", .{pid}) catch return false;
        self.parent_pid_len = parent_pid.len;
        const icon_path = if (std.mem.indexOf(u8, executable_path, ".app/Contents/MacOS/") != null)
            std.fmt.bufPrint(
                &self.status_icon_storage,
                "{s}/../Resources/assets/HarnezPadMenuBarNative.png",
                .{executable_dir},
            ) catch return false
        else
            std.fmt.bufPrint(
                &self.status_icon_storage,
                "{s}/../../assets/HarnezPadMenuBarNative.png",
                .{executable_dir},
            ) catch return false;
        self.status_icon_len = icon_path.len;
        return true;
    }

    // Packaged hosts keep the helper at Contents/Resources/harnezpad. `native dev`
    // runs from zig-out/bin, so also accept the repo-local helper locations
    // used during development before a full .app package exists.
    fn resolveHelperPath(self: *HarnezPadApp, executable_path: []const u8, executable_dir: []const u8) bool {
        const packaged = std.mem.indexOf(u8, executable_path, ".app/Contents/MacOS/") != null;
        if (packaged) {
            return self.setHelperPath("{s}/../Resources/harnezpad", executable_dir);
        }
        if (self.tryHelperPath("{s}/../Resources/harnezpad", executable_dir)) return true;
        if (self.tryHelperPath("{s}/../../dist/HarnezPad.app/Contents/Resources/harnezpad", executable_dir)) return true;
        if (self.tryHelperPath("{s}/../../harnezpad", executable_dir)) return true;
        return self.setHelperPath("{s}/../Resources/harnezpad", executable_dir);
    }

    fn tryHelperPath(self: *HarnezPadApp, comptime pattern: []const u8, executable_dir: []const u8) bool {
        const helper_path = std.fmt.bufPrint(&self.helper_path_storage, pattern, .{executable_dir}) catch return false;
        if (!helperExecutableExists(helper_path)) return false;
        self.helper_path_len = helper_path.len;
        return true;
    }

    fn setHelperPath(self: *HarnezPadApp, comptime pattern: []const u8, executable_dir: []const u8) bool {
        const helper_path = std.fmt.bufPrint(&self.helper_path_storage, pattern, .{executable_dir}) catch return false;
        self.helper_path_len = helper_path.len;
        return true;
    }

    fn helperPath(self: *const HarnezPadApp) []const u8 {
        return self.helper_path_storage[0..self.helper_path_len];
    }

    fn parentPid(self: *const HarnezPadApp) []const u8 {
        return self.parent_pid_storage[0..self.parent_pid_len];
    }

    fn statusIcon(self: *const HarnezPadApp) []const u8 {
        return self.status_icon_storage[0..self.status_icon_len];
    }

    fn start(context: *anyopaque, runtime: *native_sdk.Runtime) anyerror!void {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        self.runtime = runtime;
        menu_dispatch_app = self;
        if (!builtin.is_test and builtin.os.tag == .macos and self.native_chrome_enabled and runtime.supports(.view_surface_adoption)) {
            // `App.start` precedes scene reconciliation. Defer surface
            // adoption to the next AppKit run-loop turn, after the declared
            // native containers exist.
            try runtime.startTimer(native_chrome_timer_id, std.time.ns_per_ms, false);
        }
        try runtime.createTray(.{
            .icon_path = self.statusIcon(),
            .tooltip = "HarnezPad",
            .items = &tray_items,
        });
        if (self.helper_enabled) {
            self.startHelper() catch |err| self.setHelperError(@errorName(err));
        } else {
            self.helper_mutex.lock();
            self.helper_state = .stopped;
            self.helper_mutex.unlock();
        }
    }

    fn nativeChromeStartDrag(context: *anyopaque) callconv(.c) void {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        const runtime = self.runtime orelse return;
        runtime.options.platform.services.startWindowDrag(1) catch {};
    }

    fn dispatchMenuCommand(self: *HarnezPadApp, runtime: *native_sdk.Runtime, command: []const u8) !void {
        try self.handleCommand(runtime, .{
            .name = command,
            .source = .menu,
            .window_id = 1,
        });
    }

    fn stop(context: *anyopaque, runtime: *native_sdk.Runtime) anyerror!void {
        _ = runtime;
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        self.stopping.store(true, .release);
        const pid = self.helper_pid.load(.acquire);
        if (pid > 0) std.posix.kill(@intCast(pid), std.posix.SIG.TERM) catch {};
        if (self.helper_thread) |thread| {
            thread.join();
            self.helper_thread = null;
        }
        self.runtime = null;
    }

    fn event(context: *anyopaque, runtime: *native_sdk.Runtime, event_value: native_sdk.Event) anyerror!void {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        switch (event_value) {
            .command => |command| try self.handleCommand(runtime, command),
            .timer => |timer| {
                if (!builtin.is_test and timer.id == native_chrome_timer_id and !self.native_chrome_installed) {
                    try macos_chrome.install(runtime, self, nativeChromeStartDrag);
                    macos_app_menu.install(menuCommandDispatch);
                    self.native_chrome_installed = true;
                }
            },
            .appearance_changed => |appearance| {
                self.appearance = appearance.color_scheme;
                var detail: [96]u8 = undefined;
                const json = try std.fmt.bufPrint(
                    &detail,
                    "{{\"appearance\":\"{s}\"}}",
                    .{@tagName(appearance.color_scheme)},
                );
                try runtime.emitWindowEvent(1, "harnezpad:appearance", json);
            },
            else => {},
        }
    }

    fn handleCommand(
        self: *HarnezPadApp,
        runtime: *native_sdk.Runtime,
        command: native_sdk.CommandEvent,
    ) anyerror!void {
        const window_id = if (command.window_id == 0) 1 else command.window_id;
        if (std.mem.eql(u8, command.name, sidebar_command)) {
            _ = try self.toggleSidebar(runtime, window_id);
            return;
        }
        if (std.mem.eql(u8, command.name, open_command)) {
            try runtime.showWindow(window_id);
            return;
        }
        if (std.mem.eql(u8, command.name, quit_command)) {
            try runtime.quitApp();
            return;
        }

        const route: ?[]const u8 =
            if (std.mem.eql(u8, command.name, launch_command)) "agents"
            else if (std.mem.eql(u8, command.name, keys_command)) "keys"
            else if (std.mem.eql(u8, command.name, settings_command)) "settings"
            else if (std.mem.eql(u8, command.name, check_updates_command)) "updates"
            else null;
        if (route) |value| {
            if (!std.mem.eql(u8, value, "updates")) {
                try runtime.showWindow(window_id);
            }
            var detail: [192]u8 = undefined;
            const json = try std.fmt.bufPrint(
                &detail,
                "{{\"route\":\"{s}\",\"source\":\"{s}\"}}",
                .{ value, @tagName(command.source) },
            );
            try runtime.emitWindowEvent(window_id, "harnezpad:navigate", json);
        }
    }

    fn toggleSidebar(
        self: *HarnezPadApp,
        runtime: *native_sdk.Runtime,
        window_id: u64,
    ) anyerror!bool {
        self.sidebar_open = !self.sidebar_open;
        var detail: [40]u8 = undefined;
        const json = try std.fmt.bufPrint(
            &detail,
            "{{\"open\":{s}}}",
            .{if (self.sidebar_open) "true" else "false"},
        );
        try runtime.emitWindowEvent(window_id, "harnezpad:sidebar", json);
        return self.sidebar_open;
    }

    fn bridgeBootstrap(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        _ = invocation;
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        self.helper_mutex.lock();
        defer self.helper_mutex.unlock();
        const ready = self.helper_state == .ready;
        const url = if (ready) self.helper_url_storage[0..self.helper_url_len] else "";
        const token = if (ready) self.helper_token_storage[0..self.helper_token_len] else "";
        return std.fmt.bufPrint(
            output,
            "{{\"ready\":{s},\"helperUrl\":\"{s}\",\"helperToken\":\"{s}\",\"sidebarOpen\":{s},\"appearance\":\"{s}\"}}",
            .{
                if (ready) "true" else "false",
                url,
                token,
                if (self.sidebar_open) "true" else "false",
                @tagName(self.appearance),
            },
        );
    }

    fn bridgeToggleSidebar(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        const runtime = self.runtime orelse return error.RuntimeUnavailable;
        const open = try self.toggleSidebar(runtime, invocation.source.window_id);
        return std.fmt.bufPrint(
            output,
            "{{\"sidebarOpen\":{s}}}",
            .{if (open) "true" else "false"},
        );
    }

    const FramePayload = struct {
        x: f32,
        y: f32,
        width: f32,
        height: f32,
    };

    const DragRegionsPayload = struct {
        sidebar: FramePayload,
        content: FramePayload,
    };

    fn bridgeSetTitlebarDragRegions(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        const runtime = self.runtime orelse return error.RuntimeUnavailable;
        if (!self.native_chrome_installed) {
            return std.fmt.bufPrint(output, "{{\"accepted\":false}}", .{});
        }
        const parsed = try std.json.parseFromSlice(
            DragRegionsPayload,
            std.heap.page_allocator,
            invocation.request.payload,
            .{ .ignore_unknown_fields = true },
        );
        defer parsed.deinit();
        try macos_chrome.setDragRegions(
            runtime,
            geometry.RectF.init(
                parsed.value.sidebar.x,
                parsed.value.sidebar.y,
                parsed.value.sidebar.width,
                parsed.value.sidebar.height,
            ),
            geometry.RectF.init(
                parsed.value.content.x,
                parsed.value.content.y,
                parsed.value.content.width,
                parsed.value.content.height,
            ),
        );
        return std.fmt.bufPrint(output, "{{\"accepted\":true}}", .{});
    }

    fn bridgeQuit(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        _ = invocation;
        const self: *HarnezPadApp = @ptrCast(@alignCast(context));
        const runtime = self.runtime orelse return error.RuntimeUnavailable;
        try runtime.quitApp();
        return std.fmt.bufPrint(output, "{{\"accepted\":true}}", .{});
    }

    const DialogPayload = struct {
        title: ?[]const u8 = null,
        message: ?[]const u8 = null,
    };

    fn bridgeUpdateAlert(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        _ = context;
        const parsed = try std.json.parseFromSlice(
            DialogPayload,
            std.heap.page_allocator,
            invocation.request.payload,
            .{ .ignore_unknown_fields = true },
        );
        defer parsed.deinit();
        macos_alert.presentInformational(
            parsed.value.title orelse "HarnezPad Updates",
            parsed.value.message orelse "Update check complete.",
        );
        return std.fmt.bufPrint(output, "{{\"accepted\":true}}", .{});
    }

    fn bridgeUpdateConfirm(
        context: *anyopaque,
        invocation: native_sdk.bridge.Invocation,
        output: []u8,
    ) anyerror![]const u8 {
        _ = context;
        const parsed = try std.json.parseFromSlice(
            DialogPayload,
            std.heap.page_allocator,
            invocation.request.payload,
            .{ .ignore_unknown_fields = true },
        );
        defer parsed.deinit();
        const confirmed = macos_alert.presentConfirm(
            parsed.value.title orelse "HarnezPad Updates",
            parsed.value.message orelse "Install the downloaded update and restart HarnezPad?",
        );
        return std.fmt.bufPrint(output, "{{\"confirmed\":{s}}}", .{
            if (confirmed) "true" else "false",
        });
    }

    fn startHelper(self: *HarnezPadApp) anyerror!void {
        if (self.helper_path_len == 0 or self.parent_pid_len == 0) return error.HelperPathUnavailable;
        var child = try std.process.spawn(self.io, .{
            .argv = &.{ self.helperPath(), "serve-native", "--parent-pid", self.parentPid() },
            .stdin = .ignore,
            .stdout = .pipe,
            .stderr = .inherit,
            .pgid = 0,
        });
        errdefer child.kill(self.io);
        self.helper_pid.store(@intCast(child.id orelse return error.HelperSpawnFailed), .release);
        errdefer self.helper_pid.store(0, .release);
        const stdout_file = child.stdout orelse return error.HelperStdoutUnavailable;
        var line_storage: [16 * 1024]u8 = undefined;
        const line = try readLine(stdout_file, self.io, &line_storage);
        if (!self.acceptReadyLine(line)) return error.InvalidHelperReadyEvent;
        self.helper_thread = try std.Thread.spawn(.{}, helperMain, .{ self, child });
    }

    fn helperMain(self: *HarnezPadApp, child_value: std.process.Child) void {
        var child = child_value;
        if (child.stdout) |stdout_file| drainHelperOutput(self, stdout_file);
        _ = child.wait(self.io) catch {};
        self.helper_pid.store(0, .release);
        self.helper_mutex.lock();
        defer self.helper_mutex.unlock();
        if (self.stopping.load(.acquire)) {
            self.helper_state = .stopped;
        } else {
            self.helper_state = .failed;
            self.setHelperErrorLocked("HarnezPad helper exited");
        }
    }

    fn drainHelperOutput(self: *HarnezPadApp, stdout_file: std.Io.File) void {
        var read_buffer: [1024]u8 = undefined;
        var line_buffer: [16 * 1024]u8 = undefined;
        var line_len: usize = 0;
        while (!self.stopping.load(.acquire)) {
            const slices: [1][]u8 = .{&read_buffer};
            const count = stdout_file.readStreaming(self.io, &slices) catch break;
            if (count == 0) break;
            for (read_buffer[0..count]) |byte| {
                if (byte == '\n') {
                    line_len = 0;
                } else if (byte != '\r' and line_len < line_buffer.len) {
                    line_buffer[line_len] = byte;
                    line_len += 1;
                }
            }
        }
    }

    fn readLine(file: std.Io.File, io: std.Io, buffer: []u8) anyerror![]const u8 {
        var len: usize = 0;
        while (len < buffer.len) {
            var byte_storage: [1]u8 = undefined;
            const slices: [1][]u8 = .{&byte_storage};
            const count = try file.readStreaming(io, &slices);
            if (count == 0) return error.EndOfStream;
            const byte = byte_storage[0];
            if (byte == '\n') return buffer[0..len];
            if (byte != '\r') {
                buffer[len] = byte;
                len += 1;
            }
        }
        return error.StreamTooLong;
    }

    pub fn acceptReadyLine(self: *HarnezPadApp, line: []const u8) bool {
        const parsed = std.json.parseFromSlice(
            ReadyEvent,
            std.heap.page_allocator,
            line,
            .{ .ignore_unknown_fields = true },
        ) catch return false;
        defer parsed.deinit();
        if (!std.mem.eql(u8, parsed.value.@"type", "ready") or parsed.value.token.len == 0) return false;
        const prefix = "http://127.0.0.1:";
        if (!std.mem.startsWith(u8, parsed.value.url, prefix)) return false;
        const port = std.fmt.parseInt(u16, parsed.value.url[prefix.len..], 10) catch return false;
        if (port == 0 or
            parsed.value.url.len > self.helper_url_storage.len or
            parsed.value.token.len > self.helper_token_storage.len)
        {
            return false;
        }

        self.helper_mutex.lock();
        defer self.helper_mutex.unlock();
        @memcpy(self.helper_url_storage[0..parsed.value.url.len], parsed.value.url);
        self.helper_url_len = parsed.value.url.len;
        @memcpy(self.helper_token_storage[0..parsed.value.token.len], parsed.value.token);
        self.helper_token_len = parsed.value.token.len;
        self.helper_error_len = 0;
        self.helper_state = .ready;
        return true;
    }

    fn setHelperError(self: *HarnezPadApp, message: []const u8) void {
        self.helper_mutex.lock();
        defer self.helper_mutex.unlock();
        self.helper_state = .failed;
        self.setHelperErrorLocked(message);
    }

    fn setHelperErrorLocked(self: *HarnezPadApp, message: []const u8) void {
        const len = @min(message.len, self.helper_error_storage.len);
        @memcpy(self.helper_error_storage[0..len], message[0..len]);
        self.helper_error_len = len;
    }
};

var menu_dispatch_app: ?*HarnezPadApp = null;

fn menuCommandDispatch(command: [*:0]const u8) callconv(.c) void {
    const self = menu_dispatch_app orelse return;
    const runtime = self.runtime orelse return;
    self.dispatchMenuCommand(runtime, std.mem.span(command)) catch {};
}

fn helperExecutableExists(path: []const u8) bool {
    if (comptime builtin.os.tag != .macos) return false;
    var buffer: [std.fs.max_path_bytes:0]u8 = undefined;
    if (path.len == 0 or path.len >= buffer.len) return false;
    @memcpy(buffer[0..path.len], path);
    buffer[path.len] = 0;
    return std.c.access(buffer[0..path.len :0].ptr, std.c.X_OK) == 0;
}

pub fn main(init: std.process.Init) !void {
    var executable_storage: [std.fs.max_path_bytes]u8 = undefined;
    const executable_len = try std.process.executablePath(init.io, &executable_storage);
    const executable_path = executable_storage[0..executable_len];

    var state = HarnezPadApp{
        .env_map = init.environ_map,
        .io = init.io,
    };
    if (!state.configureLaunch(executable_path, std.posix.system.getpid())) {
        return error.HelperLaunchPathUnavailable;
    }

    try runner.runWithOptions(state.app(), .{
        .app_name = "HarnezPad",
        .window_title = "HarnezPad",
        .bundle_id = bundle_id,
        .icon_path = "assets/HarnezPadNativeIcon.png",
        .default_frame = geometry.RectF.init(0, 0, window_width, window_height),
        .restore_state = true,
        .bridge = state.bridge(),
        .builtin_bridge = .{ .enabled = true, .commands = &builtin_policies },
        .js_window_api = true,
        .security = .{
            .permissions = &app_permissions,
            .navigation = .{
                .allowed_origins = &allowed_origins,
                .external_links = .{
                    .action = .open_system_browser,
                    .allowed_urls = &.{
                        "https://gateway.example.com/*",
                        "https://github.com/harnezai/harnezpad/*",
                    },
                },
            },
        },
    }, init);
}

test "hybrid scene preserves the native window and WebView contract" {
    const window = shell_scene.windows[0];
    try std.testing.expectEqual(@as(f32, 1030), window.width);
    try std.testing.expectEqual(@as(f32, 760), window.height);
    try std.testing.expectEqual(.hidden_inset_tall, window.titlebar);
    try std.testing.expectEqual(.hide, window.close_policy);
    try std.testing.expectEqual(.webview, window.views[0].kind);
    try std.testing.expectEqualStrings(main_webview_label, window.views[0].label);
    try std.testing.expectEqual(.titlebar_accessory, window.views[1].kind);
    try std.testing.expectEqual(.icon_button, window.views[2].kind);
    try std.testing.expectEqualStrings("sidebar-toggle", window.views[2].label);
    try std.testing.expectEqualStrings(sidebar_command, window.views[2].command.?);
    try std.testing.expectEqual(@as(?f32, 84), window.views[1].x);
    try std.testing.expectEqual(@as(?f32, 122), window.views[3].x);
    try std.testing.expectEqual(@as(?f32, 114), window.views[3].width);
    try std.testing.expectEqualStrings("titlebar-drag-sidebar", window.views[3].label);
    try std.testing.expectEqualStrings("titlebar-drag-content", window.views[4].label);
    try std.testing.expect(!window.views[4].visible);
}

test "production source points at the Vite build" {
    const source_value = native_sdk.frontend.productionSource(.{
        .dist = "frontend/dist",
        .entry = "index.html",
    });
    try std.testing.expectEqual(native_sdk.WebViewSourceKind.assets, source_value.kind);
    try std.testing.expectEqualStrings("frontend/dist", source_value.asset_options.?.root_path);
}

test "helper ready event accepts only authenticated loopback sessions" {
    var env = std.process.Environ.Map.init(std.testing.allocator);
    defer env.deinit();
    var state = HarnezPadApp{ .env_map = &env, .io = std.testing.io };
    try std.testing.expect(!state.acceptReadyLine("not-json"));
    try std.testing.expect(!state.acceptReadyLine(
        "{\"type\":\"ready\",\"url\":\"https://example.com\",\"token\":\"session\"}",
    ));
    try std.testing.expect(state.acceptReadyLine(
        "{\"type\":\"ready\",\"url\":\"http://127.0.0.1:49152\",\"token\":\"session-123\"}",
    ));
    try std.testing.expectEqualStrings(
        "http://127.0.0.1:49152",
        state.helper_url_storage[0..state.helper_url_len],
    );
}

test "configureLaunch resolves packaged helper beside the host" {
    var env = std.process.Environ.Map.init(std.testing.allocator);
    defer env.deinit();
    var state = HarnezPadApp{ .env_map = &env, .io = std.testing.io };
    try std.testing.expect(state.configureLaunch(
        "/Applications/HarnezPad.app/Contents/MacOS/HarnezPad",
        42,
    ));
    try std.testing.expectEqualStrings(
        "/Applications/HarnezPad.app/Contents/MacOS/../Resources/harnezpad",
        state.helperPath(),
    );
    try std.testing.expectEqualStrings("42", state.parentPid());
}

test "configureLaunch uses unpackaged Resources fallback when no helper exists" {
    var env = std.process.Environ.Map.init(std.testing.allocator);
    defer env.deinit();
    var state = HarnezPadApp{ .env_map = &env, .io = std.testing.io };
    try std.testing.expect(state.configureLaunch(
        "/tmp/harnezpad-dev/zig-out/bin/HarnezPad",
        7,
    ));
    try std.testing.expectEqualStrings(
        "/tmp/harnezpad-dev/zig-out/bin/../Resources/harnezpad",
        state.helperPath(),
    );
}

test "bootstrap bridge returns the typed helper session contract" {
    var env = std.process.Environ.Map.init(std.testing.allocator);
    defer env.deinit();
    var state = HarnezPadApp{ .env_map = &env, .io = std.testing.io };
    var dispatcher = state.bridge();
    var output: [2048]u8 = undefined;
    const starting_response = dispatcher.dispatch(
        "{\"id\":\"0\",\"command\":\"harnezpad.bootstrap\",\"payload\":{}}",
        .{ .origin = "zero://app", .window_id = 1, .webview_label = main_webview_label },
        &output,
    );
    try std.testing.expect(std.mem.indexOf(u8, starting_response, "\"ready\":false") != null);
    try std.testing.expect(std.mem.indexOf(u8, starting_response, "\"helperUrl\":\"\"") != null);
    try std.testing.expect(std.mem.indexOf(u8, starting_response, "\"helperToken\":\"\"") != null);

    try std.testing.expect(state.acceptReadyLine(
        "{\"type\":\"ready\",\"url\":\"http://127.0.0.1:49152\",\"token\":\"session-123\"}",
    ));
    const response = dispatcher.dispatch(
        "{\"id\":\"1\",\"command\":\"harnezpad.bootstrap\",\"payload\":{}}",
        .{ .origin = "zero://app", .window_id = 1, .webview_label = main_webview_label },
        &output,
    );
    try std.testing.expect(std.mem.indexOf(u8, response, "\"ok\":true") != null);
    try std.testing.expect(std.mem.indexOf(u8, response, "\"ready\":true") != null);
    try std.testing.expect(std.mem.indexOf(u8, response, "\"helperUrl\":\"http://127.0.0.1:49152\"") != null);
    try std.testing.expect(std.mem.indexOf(u8, response, "\"helperToken\":\"session-123\"") != null);
    try std.testing.expect(std.mem.indexOf(u8, response, "\"sidebarOpen\":true") != null);
    try std.testing.expect(std.mem.indexOf(u8, response, "\"appearance\":\"light\"") != null);
}

test "trusted shell bridge toggles the sidebar" {
    const harness = try native_sdk.TestHarness().create(
        std.testing.allocator,
        .{ .size = geometry.SizeF.init(window_width, window_height) },
    );
    defer harness.destroy(std.testing.allocator);
    var env = std.process.Environ.Map.init(std.testing.allocator);
    defer env.deinit();
    var state = HarnezPadApp{
        .env_map = &env,
        .io = std.testing.io,
        .helper_enabled = false,
        .native_chrome_enabled = false,
    };
    harness.runtime.options.security.navigation.allowed_origins = &allowed_origins;
    harness.runtime.options.security.permissions = &app_permissions;
    harness.runtime.options.js_window_api = true;
    harness.runtime.options.builtin_bridge = .{ .enabled = true, .commands = &builtin_policies };
    try harness.start(state.app());

    var dispatcher = state.bridge();
    var output: [2048]u8 = undefined;
    const sidebar_response = dispatcher.dispatch(
        "{\"id\":\"1\",\"command\":\"harnezpad.sidebar.toggle\",\"payload\":{}}",
        .{ .origin = "zero://app", .window_id = 1, .webview_label = main_webview_label },
        &output,
    );
    try std.testing.expect(std.mem.indexOf(u8, sidebar_response, "\"sidebarOpen\":false") != null);
    try std.testing.expectEqualStrings("harnezpad:sidebar", harness.null_platform.lastWindowEventName());

    const denied = dispatcher.dispatch(
        "{\"id\":\"3\",\"command\":\"harnezpad.bootstrap\",\"payload\":{}}",
        .{ .origin = "https://example.com", .window_id = 1, .webview_label = "untrusted" },
        &output,
    );
    try std.testing.expect(std.mem.indexOf(u8, denied, "\"permission_denied\"") != null);
}
