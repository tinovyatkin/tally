import java.awt.GraphicsDevice;
import java.awt.GraphicsEnvironment;
import java.awt.Rectangle;
import java.awt.Robot;
import java.awt.image.BufferedImage;
import java.nio.file.Files;
import java.nio.file.Path;
import javax.imageio.ImageIO;

public final class CaptureScreen {
    private CaptureScreen() {}

    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            throw new IllegalArgumentException("usage: CaptureScreen <output.png>");
        }

        GraphicsDevice screen = GraphicsEnvironment.getLocalGraphicsEnvironment().getDefaultScreenDevice();
        var configuration = screen.getDefaultConfiguration();
        Rectangle bounds = configuration.getBounds();
        BufferedImage image = new Robot(screen).createScreenCapture(bounds);
        Path output = Path.of(args[0]).toAbsolutePath();
        Files.createDirectories(output.getParent());
        if (!ImageIO.write(image, "png", output.toFile())) {
            throw new IllegalStateException("no PNG writer available");
        }
        System.out.printf(
                "captured logical=%dx%d scale=%.2fx%.2f pixels=%dx%d%n",
                bounds.width,
                bounds.height,
                configuration.getDefaultTransform().getScaleX(),
                configuration.getDefaultTransform().getScaleY(),
                image.getWidth(),
                image.getHeight());
    }
}
