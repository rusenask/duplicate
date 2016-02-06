/* Copyright (C) Karolis Rusenas - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Karolis Rusenas <karolis.rusenas@gmail.com>, February 2016
 */
package org.golang.example.bind;

import android.app.Activity;
import android.os.Bundle;
import android.content.Context;

import android.content.Intent;
import go.pocketsv.Pocketsv;



public class MainActivity extends Activity {

    private Thread goProcess = null;

    public void startServer(final String mode) {

        final String path;
        if (android.os.Build.VERSION.SDK_INT >=android.os.Build.VERSION_CODES.LOLLIPOP){
            path = getNoBackupFilesDir().getAbsolutePath();
        } else{
            path = getFilesDir().getAbsolutePath();
        }

        goProcess = new Thread(new Runnable() {
            public void run() {
                Pocketsv.Start(mode, path);
            }
        });

        goProcess.start();
    }

    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        final Context context = this;
        setContentView(R.layout.activity_main);
        // starting server
        startServer("virtualize");

        Intent intent = new Intent(context, WebViewActivity.class);
        startActivity(intent);
    }
}
