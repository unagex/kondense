public class Main {
    public static void main(String[] args) {
        try {
            System.out.println("Application is waiting indefinitely...");
            while (true) {
                Thread.sleep(Long.MAX_VALUE);
            }
        } catch (InterruptedException e) {
            System.out.println("Application interrupted");
        }
    }
}
