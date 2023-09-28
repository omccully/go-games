package com.omccully.midtochart;

import java.io.Console;

public class Main {
    public static void main(String[] args) {
        String filePath = "";
        if(args.length != 1) {
             // Create the console object
            Console cnsl = System.console();
            filePath = cnsl.readLine("Enter mid file path: ");
        } else {
            filePath = args[0];
        }
        System.out.println("mid path: " + filePath);

        final Mid2Chart m = 
            new Mid2Chart(filePath);
        try {
            String str = m.convert();
            System.out.println(str);
        } catch(Exception e) {
            System.out.println(e.getMessage());
            System.exit(1);
        }
    }
}
