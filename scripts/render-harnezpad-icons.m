#import <AppKit/AppKit.h>

// Hugeicons TerminalIcon geometry from the shadcn Luma preset. Keeping this
// tiny vector local makes the app and menu-bar marks deterministic and avoids
// an SF Symbols runtime/build dependency.
static void DrawLumaTerminal(NSRect rect, NSColor *color) {
    [color setStroke];
    CGFloat scale = MIN(NSWidth(rect), NSHeight(rect)) / 24.0;
    CGFloat xOffset = NSMidX(rect) - 12.0 * scale;
    CGFloat yTop = NSMidY(rect) + 12.0 * scale;
    NSPoint (^point)(CGFloat, CGFloat) = ^NSPoint(CGFloat x, CGFloat y) {
        return NSMakePoint(xOffset + x * scale, yTop - y * scale);
    };
    NSBezierPath *glyph = [NSBezierPath bezierPath];
    glyph.lineWidth = MAX(1.0, 1.5 * scale);
    glyph.lineCapStyle = NSLineCapStyleRound;
    glyph.lineJoinStyle = NSLineJoinStyleRound;
    [glyph moveToPoint:point(4.0, 5.0)];
    [glyph curveToPoint:point(10.0, 11.0)
          controlPoint1:point(4.0, 5.0)
          controlPoint2:point(10.0, 9.42)];
    [glyph curveToPoint:point(4.0, 17.0)
          controlPoint1:point(10.0, 12.58)
          controlPoint2:point(4.0, 17.0)];
    [glyph moveToPoint:point(12.0, 19.0)];
    [glyph lineToPoint:point(20.0, 19.0)];
    [glyph stroke];
}

static BOOL WritePNG(NSString *path, NSInteger pixels, BOOL appIcon) {
    NSBitmapImageRep *bitmap = [[NSBitmapImageRep alloc]
        initWithBitmapDataPlanes:NULL
                      pixelsWide:pixels
                      pixelsHigh:pixels
                   bitsPerSample:8
                 samplesPerPixel:4
                        hasAlpha:YES
                        isPlanar:NO
                  colorSpaceName:NSCalibratedRGBColorSpace
                     bytesPerRow:0
                    bitsPerPixel:0];
    if (!bitmap) return NO;
    bitmap.size = NSMakeSize(pixels, pixels);

    [NSGraphicsContext saveGraphicsState];
    NSGraphicsContext *context = [NSGraphicsContext graphicsContextWithBitmapImageRep:bitmap];
    [NSGraphicsContext setCurrentContext:context];
    context.imageInterpolation = NSImageInterpolationHigh;
    [[NSColor clearColor] setFill];
    NSRectFill(NSMakeRect(0, 0, pixels, pixels));

    NSColor *glyphColor = appIcon ? [NSColor whiteColor] : [NSColor blackColor];
    CGFloat inset = appIcon ? pixels * 0.08 : pixels * 0.08;
    if (appIcon) {
        NSRect backgroundRect = NSInsetRect(NSMakeRect(0, 0, pixels, pixels), inset, inset);
        NSBezierPath *background = [NSBezierPath bezierPathWithRoundedRect:backgroundRect
                                                                   xRadius:pixels * 0.22
                                                                   yRadius:pixels * 0.22];
        [[NSColor colorWithCalibratedRed:0.075 green:0.075 blue:0.085 alpha:1.0] setFill];
        [background fill];
        inset = pixels * 0.25;
    }

    NSRect glyphRect = NSInsetRect(NSMakeRect(0, 0, pixels, pixels), inset, inset);
    DrawLumaTerminal(glyphRect, glyphColor);
    [NSGraphicsContext restoreGraphicsState];

    NSData *png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 2) {
            fprintf(stderr, "usage: render-harnezpad-icons <output-directory>\n");
            return 2;
        }
        NSString *output = [NSString stringWithUTF8String:argv[1]];
        NSString *iconset = [output stringByAppendingPathComponent:@"HarnezPad.iconset"];
        NSError *error = nil;
        if (![[NSFileManager defaultManager] createDirectoryAtPath:iconset
                                       withIntermediateDirectories:YES
                                                        attributes:nil
                                                             error:&error]) {
            fprintf(stderr, "%s\n", error.localizedDescription.UTF8String);
            return 1;
        }

        NSDictionary<NSString *, NSNumber *> *sizes = @{
            @"icon_16x16.png": @16,
            @"icon_16x16@2x.png": @32,
            @"icon_32x32.png": @32,
            @"icon_32x32@2x.png": @64,
            @"icon_128x128.png": @128,
            @"icon_128x128@2x.png": @256,
            @"icon_256x256.png": @256,
            @"icon_256x256@2x.png": @512,
            @"icon_512x512.png": @512,
            @"icon_512x512@2x.png": @1024,
        };
        for (NSString *name in sizes) {
            if (!WritePNG([iconset stringByAppendingPathComponent:name], sizes[name].integerValue, YES)) {
                fprintf(stderr, "failed to write %s\n", name.UTF8String);
                return 1;
            }
        }
        if (!WritePNG([output stringByAppendingPathComponent:@"HarnezPadNativeIcon.png"], 1024, YES) ||
            !WritePNG([output stringByAppendingPathComponent:@"HarnezPadMenuBarNative.png"], 18, NO) ||
            !WritePNG([output stringByAppendingPathComponent:@"HarnezPadMenuBarNative@2x.png"], 36, NO)) {
            fprintf(stderr, "failed to write HarnezPad icon assets\n");
            return 1;
        }

    }
    return 0;
}
