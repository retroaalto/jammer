using Raylib_cs;

namespace jammer
{
    public class Play
    {
        static string path = "";
        public static void PlaySong(string[] songs, int Currentindex)
        {
            // check if file is a local
            if (File.Exists(songs[Currentindex]))
            {
                // id related to local file path, convert to absolute path
                path = Path.GetFullPath(songs[Currentindex]);
            }
            else if (URL.IsValidSoundcloudSong(songs[Currentindex]))
            {
                // id related to url, download and convert to absolute path
                path = Download.DownloadSong(songs[Currentindex]);
            }
            else if (URL.IsValidYoutubeSong(songs[Currentindex]))
            {
                // id related to url, download and convert to absolute path
                path = Download.DownloadSong(songs[Currentindex]);
            }
            else
            {
                Console.WriteLine("Invalid url");
                return;
            }
            Utils.currentSong = path;
            Utils.currentSongIndex = Currentindex;
            PlayPath();
        }

        static void PlayPath() {
            // play song
            Raylib.InitAudioDevice();
            Raylib.SetMasterVolume(0.5f);
            // Utils.currentMusic = Raylib.LoadSound(path);
            // Raylib.PlaySound(Utils.currentMusic);
            Utils.currentMusic = Raylib.LoadMusicStream(path);
            Console.WriteLine("Playing music: " + path);
            Raylib.PlayMusicStream(Utils.currentMusic);
        }

        public static void PauseSong()
        {
            Raylib.PauseMusicStream(Utils.currentMusic);
        }

        public static void ResumeSong()
        {
            Raylib.ResumeMusicStream(Utils.currentMusic);
        }
    }
}