#import "sfsymbol_darwin.h"
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

void *HarnezPadSFSymbolPNG(const char *name, int pointSize, size_t *outLen) {
    if (!name || !outLen || pointSize < 8 || pointSize > 64) {
        return NULL;
    }
    *outLen = 0;
    if (@available(macOS 11.0, *)) {
        NSString *symbolName = [NSString stringWithUTF8String:name];
        if (symbolName.length == 0) {
            return NULL;
        }

        NSImageSymbolConfiguration *config =
            [NSImageSymbolConfiguration configurationWithPointSize:pointSize
                                                              weight:NSFontWeightRegular
                                                               scale:NSImageSymbolScaleMedium];
        NSImage *symbol = [NSImage imageWithSystemSymbolName:symbolName accessibilityDescription:nil];
        if (!symbol) {
            return NULL;
        }
        symbol = [symbol imageWithSymbolConfiguration:config];
        if (!symbol) {
            return NULL;
        }

        // Render into a retina bitmap so template glyphs keep crisp edges
        // (lockFocus often produces muddy/filled squares for some symbols).
        const CGFloat scale = 2.0;
        const NSInteger pixels = (NSInteger)llround((CGFloat)pointSize * scale);
        NSBitmapImageRep *rep = [[NSBitmapImageRep alloc]
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
        if (!rep) {
            return NULL;
        }
        rep.size = NSMakeSize(pointSize, pointSize);

        [NSGraphicsContext saveGraphicsState];
        NSGraphicsContext *ctx = [NSGraphicsContext graphicsContextWithBitmapImageRep:rep];
        if (!ctx) {
            [NSGraphicsContext restoreGraphicsState];
            return NULL;
        }
        [NSGraphicsContext setCurrentContext:ctx];
        ctx.imageInterpolation = NSImageInterpolationHigh;

        [[NSColor clearColor] set];
        NSRectFill(NSMakeRect(0, 0, pointSize, pointSize));
        [symbol drawInRect:NSMakeRect(0, 0, pointSize, pointSize)
                  fromRect:NSZeroRect
                 operation:NSCompositingOperationSourceOver
                  fraction:1.0
            respectFlipped:YES
                     hints:@{NSImageHintInterpolation : @(NSImageInterpolationHigh)}];
        [NSGraphicsContext restoreGraphicsState];

        // Force template glyphs to opaque black on transparent for CSS invert.
        unsigned char *data = [rep bitmapData];
        if (data) {
            NSInteger total = pixels * pixels;
            for (NSInteger i = 0; i < total; i++) {
                unsigned char *px = data + i * 4;
                unsigned char alpha = px[3];
                if (alpha == 0) {
                    px[0] = px[1] = px[2] = 0;
                    continue;
                }
                // Any covered pixel becomes black with preserved coverage.
                px[0] = px[1] = px[2] = 0;
                px[3] = alpha;
            }
        }

        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        if (!png || png.length == 0) {
            return NULL;
        }
        void *buf = malloc(png.length);
        if (!buf) {
            return NULL;
        }
        memcpy(buf, png.bytes, png.length);
        *outLen = png.length;
        return buf;
    }
    return NULL;
}
