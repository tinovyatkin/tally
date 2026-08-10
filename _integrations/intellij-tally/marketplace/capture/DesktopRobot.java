import java.awt.Robot;
import java.awt.Toolkit;
import java.awt.datatransfer.StringSelection;
import java.awt.event.InputEvent;
import java.awt.event.KeyEvent;
import java.lang.reflect.Field;
import java.util.Locale;

public final class DesktopRobot {
    private DesktopRobot() {}

    public static void main(String[] args) throws Exception {
        Robot robot = new Robot();
        robot.setAutoDelay(80);
        switch (args.length == 0 ? "" : args[0]) {
            case "click" -> click(robot, args);
            case "press" -> press(robot, args);
            case "type" -> type(robot, args);
            default -> throw new IllegalArgumentException(
                    "usage: DesktopRobot click <x> <y> | press <key...> | type <text>");
        }
    }

    private static void click(Robot robot, String[] args) {
        if (args.length != 3) {
            throw new IllegalArgumentException("usage: DesktopRobot click <x> <y>");
        }
        robot.mouseMove(Integer.parseInt(args[1]), Integer.parseInt(args[2]));
        robot.delay(300);
        robot.mousePress(InputEvent.BUTTON1_DOWN_MASK);
        robot.mouseRelease(InputEvent.BUTTON1_DOWN_MASK);
    }

    private static void press(Robot robot, String[] args) throws Exception {
        if (args.length < 2) {
            throw new IllegalArgumentException("usage: DesktopRobot press <key...>");
        }
        int[] keys = new int[args.length - 1];
        for (int i = 1; i < args.length; i++) {
            keys[i - 1] = keyCode(args[i]);
            robot.keyPress(keys[i - 1]);
        }
        for (int i = keys.length - 1; i >= 0; i--) {
            robot.keyRelease(keys[i]);
        }
    }

    private static void type(Robot robot, String[] args) {
        if (args.length != 2) {
            throw new IllegalArgumentException("usage: DesktopRobot type <text>");
        }
        Toolkit.getDefaultToolkit().getSystemClipboard().setContents(new StringSelection(args[1]), null);
        robot.keyPress(KeyEvent.VK_CONTROL);
        robot.keyPress(KeyEvent.VK_V);
        robot.keyRelease(KeyEvent.VK_V);
        robot.keyRelease(KeyEvent.VK_CONTROL);
    }

    private static int keyCode(String name) throws Exception {
        String normalized = name.toUpperCase(Locale.ROOT);
        if ("CTRL".equals(normalized)) {
            normalized = "CONTROL";
        }
        Field field = KeyEvent.class.getField("VK_" + normalized);
        return field.getInt(null);
    }
}
