import java.util.prefs.Preferences;

public final class SeedJetBrainsProfile {
    private SeedJetBrainsProfile() {}

    public static void main(String[] args) throws Exception {
        Preferences agreements = Preferences.userRoot().node("jetbrains/privacy_policy");
        agreements.put("eua_accepted_version", "2.0");
        agreements.flush();
    }
}
