/* Copyright (C) Karolis Rusenas - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Karolis Rusenas <karolis.rusenas@gmail.com>, February 2016
 */
package org.golang.example.bind;

import android.app.Activity;
import android.net.wifi.WifiInfo;
import android.os.Bundle;
import android.content.Context;

import android.content.Intent;
import go.duplicate.Duplicate;

import android.net.wifi.WifiManager;
import android.util.Log;

import java.util.Locale;


public class MainActivity extends Activity {

    /**
     * Get the IP of current Wi-Fi connection
     * @return IP as string
     */
    private String getIP() {
        try {
            WifiManager wifiManager = (WifiManager) getSystemService(WIFI_SERVICE);
            WifiInfo wifiInfo = wifiManager.getConnectionInfo();
            int ipAddress = wifiInfo.getIpAddress();
            return String.format(Locale.getDefault(), "%d.%d.%d.%d",
                    (ipAddress & 0xff), (ipAddress >> 8 & 0xff),
                    (ipAddress >> 16 & 0xff), (ipAddress >> 24 & 0xff));
        } catch (Exception ex) {
            Log.e("wifiAccessFailed", ex.getMessage());
            return null;
        }
    }


    private Thread goProcess = null;

    /**
     * Starts Duplicate server
     * @param mode string
     */
    public void startServer(final String mode) {

        final String path;
        if (android.os.Build.VERSION.SDK_INT >=android.os.Build.VERSION_CODES.LOLLIPOP){
            path = getNoBackupFilesDir().getAbsolutePath();
        } else{
            path = getFilesDir().getAbsolutePath();
        }

        final String ipAddress;
        ipAddress = getIP();

        goProcess = new Thread(new Runnable() {
            public void run() {
                Duplicate.Start(mode, path, ipAddress);
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
