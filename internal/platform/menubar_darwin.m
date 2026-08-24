#import "menubar_darwin.h"
#import <Cocoa/Cocoa.h>

@class HarnezPadStatusController;
@class HarnezPadToolbarDelegate;

@interface HarnezPadDragRegionView : NSView
@end

@implementation HarnezPadDragRegionView

- (BOOL)mouseDownCanMoveWindow {
    return YES;
}

- (void)mouseDown:(NSEvent *)event {
    [self.window performWindowDragWithEvent:event];
}

@end

@interface HarnezPadWindowDelegateProxy : NSObject <NSWindowDelegate>
@property(nonatomic, assign) HarnezPadStatusController *controller;
@property(nonatomic, strong) id originalDelegate;
@end

@interface HarnezPadApplicationDelegateProxy : NSObject <NSApplicationDelegate>
@property(nonatomic, assign) HarnezPadStatusController *controller;
@property(nonatomic, strong) id originalDelegate;
@property(nonatomic, assign) BOOL systemShutdownInProgress;
@end

@interface HarnezPadStatusController : NSObject
@property(nonatomic, strong) NSStatusItem *statusItem;
@property(nonatomic, assign) NSWindow *window;
@property(nonatomic, strong) HarnezPadWindowDelegateProxy *windowProxy;
@property(nonatomic, strong) HarnezPadApplicationDelegateProxy *applicationProxy;
@property(nonatomic, strong) NSMenu *originalMainMenu;
@property(nonatomic, strong) HarnezPadDragRegionView *dragRegion;
- (instancetype)initWithWindow:(NSWindow *)window;
- (void)openWindow;
- (void)hideWindow;
- (void)positionTrafficLights;
- (void)restoreDelegates;
@end

@implementation HarnezPadWindowDelegateProxy

- (BOOL)windowShouldClose:(id)sender {
    [self.controller hideWindow];
    return NO;
}

- (void)windowWillClose:(NSNotification *)notification {
    if ([self.originalDelegate respondsToSelector:@selector(windowWillClose:)]) {
        [self.originalDelegate windowWillClose:notification];
    }
}

- (void)windowDidBecomeKey:(NSNotification *)notification {
    [self.controller positionTrafficLights];
    if ([self.originalDelegate respondsToSelector:@selector(windowDidBecomeKey:)]) {
        [self.originalDelegate windowDidBecomeKey:notification];
    }
}

- (void)windowDidResize:(NSNotification *)notification {
    [self.controller positionTrafficLights];
    if ([self.originalDelegate respondsToSelector:@selector(windowDidResize:)]) {
        [self.originalDelegate windowDidResize:notification];
    }
}

- (BOOL)respondsToSelector:(SEL)selector {
    return [super respondsToSelector:selector] ||
           [self.originalDelegate respondsToSelector:selector];
}

- (id)forwardingTargetForSelector:(SEL)selector {
    if ([self.originalDelegate respondsToSelector:selector]) {
        return self.originalDelegate;
    }
    return [super forwardingTargetForSelector:selector];
}

@end

@implementation HarnezPadApplicationDelegateProxy

- (instancetype)init {
    self = [super init];
    if (self) {
        [[[NSWorkspace sharedWorkspace] notificationCenter]
            addObserver:self
               selector:@selector(systemWillPowerOff:)
                   name:NSWorkspaceWillPowerOffNotification
                 object:nil];
    }
    return self;
}

- (void)dealloc {
    [[[NSWorkspace sharedWorkspace] notificationCenter] removeObserver:self];
    [super dealloc];
}

- (void)systemWillPowerOff:(NSNotification *)notification {
    self.systemShutdownInProgress = YES;
}

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
    if (self.systemShutdownInProgress) {
        return NSTerminateNow;
    }
    harnezpadQuit();
    return NSTerminateCancel;
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender
                    hasVisibleWindows:(BOOL)hasVisibleWindows {
    [self.controller openWindow];
    return YES;
}

- (BOOL)respondsToSelector:(SEL)selector {
    return [super respondsToSelector:selector] ||
           [self.originalDelegate respondsToSelector:selector];
}

- (id)forwardingTargetForSelector:(SEL)selector {
    if ([self.originalDelegate respondsToSelector:selector]) {
        return self.originalDelegate;
    }
    return [super forwardingTargetForSelector:selector];
}

@end

@implementation HarnezPadStatusController

- (instancetype)initWithWindow:(NSWindow *)window {
    self = [super init];
    if (!self) {
        return nil;
    }

    self.window = window;
    window.title = @"HarnezPad";
    window.titleVisibility = NSWindowTitleHidden;
    window.titlebarAppearsTransparent = YES;
    window.styleMask |= NSWindowStyleMaskFullSizeContentView;
    window.movableByWindowBackground = YES;
    NSView *contentView = window.contentView;
    NSView *windowFrameView = contentView.superview ?: contentView;
    NSRect frameBounds = windowFrameView.bounds;
    // Start after the sidebar + HTML toggle so the ghost control stays clickable.
    self.dragRegion = [[HarnezPadDragRegionView alloc] initWithFrame:NSMakeRect(
        236.0, NSHeight(frameBounds) - 40.0,
        MAX(0.0, NSWidth(frameBounds) - 236.0), 40.0)];
    self.dragRegion.autoresizingMask = NSViewWidthSizable | NSViewMinYMargin;
    [windowFrameView addSubview:self.dragRegion
                     positioned:NSWindowAbove
                     relativeTo:nil];

    self.windowProxy = [[HarnezPadWindowDelegateProxy alloc] init];
    self.windowProxy.controller = self;
    self.windowProxy.originalDelegate = window.delegate;
    window.delegate = self.windowProxy;

    self.applicationProxy = [[HarnezPadApplicationDelegateProxy alloc] init];
    self.applicationProxy.controller = self;
    self.applicationProxy.originalDelegate = NSApp.delegate;
    NSApp.delegate = self.applicationProxy;

    self.originalMainMenu = NSApp.mainMenu;
    NSMenu *mainMenu = [[NSMenu alloc] initWithTitle:@"Main Menu"];
    NSMenuItem *appMenuItem = [[NSMenuItem alloc] initWithTitle:@"HarnezPad"
                                                        action:nil
                                                 keyEquivalent:@""];
    NSMenu *appMenu = [[NSMenu alloc] initWithTitle:@"HarnezPad"];
    NSMenuItem *aboutItem = [[NSMenuItem alloc] initWithTitle:@"About HarnezPad"
                                                       action:@selector(showAbout:)
                                                keyEquivalent:@""];
    aboutItem.target = self;
    [appMenu addItem:aboutItem];
    [appMenu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *appSettingsItem = [[NSMenuItem alloc] initWithTitle:@"Settings…"
                                                            action:@selector(openSettings:)
                                                     keyEquivalent:@","];
    appSettingsItem.target = self;
    [appMenu addItem:appSettingsItem];
    NSMenuItem *appUpdatesItem = [[NSMenuItem alloc] initWithTitle:@"Check for Updates…"
                                                           action:@selector(checkForUpdates:)
                                                    keyEquivalent:@""];
    appUpdatesItem.target = self;
    [appMenu addItem:appUpdatesItem];
    [appMenu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *hideItem = [[NSMenuItem alloc] initWithTitle:@"Hide HarnezPad"
                                                       action:@selector(hide:)
                                                keyEquivalent:@"h"];
    hideItem.target = NSApp;
    [appMenu addItem:hideItem];
    NSMenuItem *hideOthersItem = [[NSMenuItem alloc] initWithTitle:@"Hide Others"
                                                              action:@selector(hideOtherApplications:)
                                                       keyEquivalent:@"h"];
    hideOthersItem.keyEquivalentModifierMask = NSEventModifierFlagCommand | NSEventModifierFlagOption;
    hideOthersItem.target = NSApp;
    [appMenu addItem:hideOthersItem];
    NSMenuItem *showAllItem = [[NSMenuItem alloc] initWithTitle:@"Show All"
                                                           action:@selector(unhideAllApplications:)
                                                    keyEquivalent:@""];
    showAllItem.target = NSApp;
    [appMenu addItem:showAllItem];
    [appMenu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *appQuitItem = [[NSMenuItem alloc] initWithTitle:@"Quit HarnezPad"
                                                        action:@selector(quitHarnezPad:)
                                                 keyEquivalent:@"q"];
    appQuitItem.target = self;
    [appMenu addItem:appQuitItem];
    appMenuItem.submenu = appMenu;
    [mainMenu addItem:appMenuItem];

    NSMenuItem *editMenuItem = [[NSMenuItem alloc] initWithTitle:@"Edit"
                                                         action:nil
                                                  keyEquivalent:@""];
    NSMenu *editMenu = [[NSMenu alloc] initWithTitle:@"Edit"];
    [editMenu addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
    [editMenu addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
    [editMenu addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
    [editMenu addItem:[NSMenuItem separatorItem]];
    [editMenu addItemWithTitle:@"Select All"
                        action:@selector(selectAll:)
                 keyEquivalent:@"a"];
    editMenuItem.submenu = editMenu;
    [mainMenu addItem:editMenuItem];

    NSMenuItem *windowMenuItem = [[NSMenuItem alloc] initWithTitle:@"Window"
                                                              action:nil
                                                       keyEquivalent:@""];
    NSMenu *windowMenu = [[NSMenu alloc] initWithTitle:@"Window"];
    NSMenuItem *newWindowItem = [[NSMenuItem alloc] initWithTitle:@"New Window"
                                                             action:@selector(openWindow:)
                                                      keyEquivalent:@"n"];
    newWindowItem.target = self;
    [windowMenu addItem:newWindowItem];
    [windowMenu addItem:[NSMenuItem separatorItem]];
    [windowMenu addItemWithTitle:@"Minimize" action:@selector(performMiniaturize:) keyEquivalent:@"m"];
    [windowMenu addItemWithTitle:@"Zoom" action:@selector(performZoom:) keyEquivalent:@""];
    windowMenuItem.submenu = windowMenu;
    [mainMenu addItem:windowMenuItem];

    NSMenuItem *helpMenuItem = [[NSMenuItem alloc] initWithTitle:@"Help"
                                                            action:nil
                                                     keyEquivalent:@""];
    NSMenu *helpMenu = [[NSMenu alloc] initWithTitle:@"Help"];
    NSMenuItem *helpItem = [[NSMenuItem alloc] initWithTitle:@"HarnezPad Help"
                                                        action:@selector(showHelp:)
                                                 keyEquivalent:@"?"];
    helpItem.target = self;
    [helpMenu addItem:helpItem];
    helpMenuItem.submenu = helpMenu;
    [mainMenu addItem:helpMenuItem];

    NSApp.mainMenu = mainMenu;

    NSMenu *menu = [[NSMenu alloc] init];

    NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"Open HarnezPad"
                                                     action:@selector(openWindow:)
                                              keyEquivalent:@""];
    openItem.target = self;
    [menu addItem:openItem];

    NSMenuItem *settingsItem = [[NSMenuItem alloc] initWithTitle:@"Settings…"
                                                         action:@selector(openSettings:)
                                                  keyEquivalent:@","];
    settingsItem.target = self;
    [menu addItem:settingsItem];

    NSMenuItem *updatesItem = [[NSMenuItem alloc] initWithTitle:@"Check for Updates…"
                                                           action:@selector(checkForUpdates:)
                                                    keyEquivalent:@""];
    updatesItem.target = self;
    [menu addItem:updatesItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit HarnezPad"
                                                     action:@selector(quitHarnezPad:)
                                              keyEquivalent:@""];
    quitItem.target = self;
    [menu addItem:quitItem];

    self.statusItem = [[NSStatusBar systemStatusBar]
        statusItemWithLength:NSSquareStatusItemLength];
    self.statusItem.visible = YES;
    self.statusItem.menu = menu;
    self.statusItem.button.toolTip = @"HarnezPad";

    NSImage *menuImage = [[NSBundle mainBundle] imageForResource:@"HarnezPadMenuBar"];
    if (!menuImage) {
        NSString *basePath = [[[NSFileManager defaultManager]
            currentDirectoryPath] stringByAppendingPathComponent:@"assets"];
        menuImage = [[NSImage alloc] initWithSize:NSMakeSize(18, 18)];
        NSString *path1x = [basePath stringByAppendingPathComponent:@"HarnezPadMenuBarNative.png"];
        NSString *path2x = [basePath stringByAppendingPathComponent:@"HarnezPadMenuBarNative@2x.png"];
        NSData *data1x = [NSData dataWithContentsOfFile:path1x];
        NSData *data2x = [NSData dataWithContentsOfFile:path2x];
        if (data1x) {
            NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithData:data1x];
            if (rep) {
                [menuImage addRepresentation:rep];
            }
        }
        if (data2x) {
            NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithData:data2x];
            if (rep) {
                rep.size = NSMakeSize(18, 18);
                [menuImage addRepresentation:rep];
            }
        }
        if (menuImage.representations.count == 0) {
            menuImage = nil;
        }
    }
    if (menuImage) {
        menuImage.template = YES;
        menuImage.size = NSMakeSize(18, 18);
        self.statusItem.button.image = menuImage;
        self.statusItem.button.imageScaling = NSImageScaleProportionallyDown;
    }

    NSImage *appImage = [[NSBundle mainBundle] imageForResource:@"HarnezPad"];
    if (!appImage) {
        NSString *developmentPath = [[[NSFileManager defaultManager]
            currentDirectoryPath] stringByAppendingPathComponent:
                @"assets/HarnezPadNative.icns"];
        appImage = [[NSImage alloc] initWithContentsOfFile:developmentPath];
    }
    if (appImage) {
        NSApp.applicationIconImage = appImage;
    }

    dispatch_async(dispatch_get_main_queue(), ^{
        [self positionTrafficLights];
    });

    return self;
}

- (void)positionTrafficLights {
    NSButton *trafficLights[] = {
        [self.window standardWindowButton:NSWindowCloseButton],
        [self.window standardWindowButton:NSWindowMiniaturizeButton],
        [self.window standardWindowButton:NSWindowZoomButton],
    };
    NSButton *closeButton = trafficLights[0];
    if (!closeButton || !closeButton.superview) {
        return;
    }

    const CGFloat leftInset = 16.0;
    const CGFloat topInset = 14.0;
    const CGFloat deltaX = leftInset - NSMinX(closeButton.frame);
    const CGFloat targetY = NSHeight(closeButton.superview.bounds)
        - topInset - NSHeight(closeButton.frame);
    const CGFloat deltaY = targetY - NSMinY(closeButton.frame);
    for (NSUInteger index = 0; index < 3; index++) {
        NSButton *button = trafficLights[index];
        if (!button) {
            continue;
        }
        NSRect frame = button.frame;
        frame.origin.x += deltaX;
        frame.origin.y += deltaY;
        button.frame = frame;
    }
}

- (void)openWindow:(id)sender {
    [self openWindow];
}

- (void)openSettings:(id)sender {
    [self openWindow];
    harnezpadShowSettings();
}

- (void)showAbout:(id)sender {
    harnezpadShowAbout();
}

- (void)showHelp:(id)sender {
    [self openWindow];
    harnezpadShowHelp();
}

- (void)checkForUpdates:(id)sender {
    harnezpadCheckForUpdates();
}

- (void)quitHarnezPad:(id)sender {
    harnezpadQuit();
}

- (void)openWindow {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp unhide:nil];
    [self.window makeKeyAndOrderFront:nil];
    [self positionTrafficLights];
    [NSApp activateIgnoringOtherApps:YES];
}

- (void)hideWindow {
    [NSApp hide:nil];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

- (void)restoreDelegates {
    if (self.window.delegate == self.windowProxy) {
        self.window.delegate = self.windowProxy.originalDelegate;
    }
    if (NSApp.delegate == self.applicationProxy) {
        NSApp.delegate = self.applicationProxy.originalDelegate;
    }
    if (NSApp.mainMenu != self.originalMainMenu) {
        NSApp.mainMenu = self.originalMainMenu;
    }
}

@end

void harnezpadPresentAbout(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString *appVersion = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"CFBundleShortVersionString"];
        if (!appVersion || appVersion.length == 0) {
            appVersion = @"dev";
        }
        NSDictionary *options = @{
            NSAboutPanelOptionApplicationName: @"HarnezPad",
            NSAboutPanelOptionApplicationVersion: appVersion,
        };
        [NSApp orderFrontStandardAboutPanelWithOptions:options];
        [NSApp activateIgnoringOtherApps:YES];
    });
}

void harnezpadPresentUpdateAlert(const char *title, const char *message) {
    NSString *alertTitle = title ? [NSString stringWithUTF8String:title] : @"HarnezPad Updates";
    NSString *alertMessage = message ? [NSString stringWithUTF8String:message] : @"Update check complete.";
    dispatch_async(dispatch_get_main_queue(), ^{
        NSAlert *alert = [[NSAlert alloc] init];
        alert.messageText = alertTitle;
        alert.informativeText = alertMessage;
        alert.alertStyle = NSAlertStyleInformational;
        [alert addButtonWithTitle:@"OK"];
        [alert runModal];
    });
}

static int harnezpadRunUpdateConfirm(NSString *alertTitle, NSString *alertMessage) {
    NSAlert *alert = [[NSAlert alloc] init];
    alert.messageText = alertTitle;
    alert.informativeText = alertMessage;
    alert.alertStyle = NSAlertStyleInformational;
    [alert addButtonWithTitle:@"Install and Restart"];
    [alert addButtonWithTitle:@"Cancel"];
    return [alert runModal] == NSAlertFirstButtonReturn ? 1 : 0;
}

int harnezpadPresentUpdateConfirm(const char *title, const char *message) {
    NSString *alertTitle = title ? [NSString stringWithUTF8String:title] : @"HarnezPad Updates";
    NSString *alertMessage = message ? [NSString stringWithUTF8String:message] : @"Install the downloaded update and restart HarnezPad?";
    __block int result = 0;
    void (^present)(void) = ^{
        result = harnezpadRunUpdateConfirm(alertTitle, alertMessage);
    };
    if ([NSThread isMainThread]) {
        present();
    } else {
        dispatch_sync(dispatch_get_main_queue(), present);
    }
    return result;
}

static HarnezPadStatusController *harnezpadStatusController = nil;

void harnezpadInstallMenuBar(void *windowPointer) {
    NSWindow *window = (__bridge NSWindow *)windowPointer;
    if (!window || harnezpadStatusController) {
        return;
    }
    harnezpadStatusController = [[HarnezPadStatusController alloc] initWithWindow:window];
}

void harnezpadUninstallMenuBar(void) {
    if (!harnezpadStatusController) {
        return;
    }
    [harnezpadStatusController restoreDelegates];
    [[NSStatusBar systemStatusBar] removeStatusItem:harnezpadStatusController.statusItem];
    harnezpadStatusController = nil;
}

