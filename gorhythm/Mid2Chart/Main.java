public class Main {
    public static void main(String[] args) {
        final Mid2Chart m = 
            new Mid2Chart(args[0]);
        try {
            String str = m.convert();
            System.out.println(str);
        } catch(Exception e) {
            System.out.println(e.getMessage());
            System.exit(1);
        }
    }
}
