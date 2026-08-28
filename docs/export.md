# Preparing an Ente Auth export

If your Authenticator app is other that @ENTE_AUTH@, install it first. But
before that, check @IMPORT_LIST@ to make sure your Authenticator app is in the
list. **If your Authenticator is not in the import list, then you cannot use @ME@
without some work.**. Please look at FAQ for workaround.

To find the installed Ente Auth version on an iPhone, open **Settings**, then
go to **General > iPhone Storage > Ente Auth**. The version is displayed below
the app name.

The following screenshot shows the import formats supported by Ente Auth 4.4.25.

<p align="center">
    <img 
        src="images/ente_import_framed.png" 
        width="40%"
        alt="Ente Auth import formats"
    /> 
</p>



@ME@ requires an encrypted export file from @ENTE_AUTH@ and the
password used to create it.

<table>
    <tr>
        <td align="center">
            <img src="images/authenticators_framed.png" width="313"
            alt="Authenticator apps"><br>
        </td>
        <td align="center">
            <img src="images/ente_settings_framed.png"
            alt="Ente Settings"
            width="313" alt="iPhone 17 Pro frame"><br>
        </td>
    </tr>
</table>


* Open @ENTE_AUTH@, then tap the hamburger menu icon in the upper-left corner to open
  Settings.

* Tap **Data**.

After importing your codes, return to the **Data** screen and export your
secrets to an encrypted JSON file.

## Export codes

The following screenshots show how to create an encrypted export on an
iPhone using Ente Auth 4.4.25. Menu names and locations may differ in other
versions.

<table>
    <tr>
        <td align="center">
            <img src="images/ente_export_framed.png" width="313"
            alt="Ente export"><br>
        </td>
        <td align="center">
            <img src="images/ente_export_pass_framed.png"
            alt="Ente pass"
            width="313" alt="iPhone 17 Pro frame"><br>
        </td>
    </tr>
</table>

* Tap **Export codes**, then select **Encrypted**.

* Enter a strong password, then tap **Save**.

Save the file on your phone, then transfer it to the computer where you will
run @ME@. For example, you can save it to local iPhone storage and
use AirDrop to transfer it to a Mac. Use the equivalent export and transfer
steps for @ENTE_AUTH@ on other platforms.

Use @ME@ to display live 2FA codes or show QR codes for importing
entries into another authenticator app. See [Usage](#usage) for the available
commands.

Create a new encrypted export whenever you add, remove, or change a 2FA entry,
and keep backup copies in safe locations.

_The screenshots are framed with @IPHONE_FRAMEIT@_
