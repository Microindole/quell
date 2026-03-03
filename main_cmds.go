package main

import (
	"quell/internal/crawler"
	"quell/internal/downloader"

	tea "github.com/charmbracelet/bubbletea"
)

type fetchResultMsg struct {
	videos []crawler.BiliVideoMeta
	total  int
	err    error
}

type downloadResultMsg struct {
	bvid string
	err  error
}

func fetchVideosCmd(uid string, page int) tea.Cmd {
	return func() tea.Msg {
		videos, total, err := crawler.GetUserVideos(uid, page, 30)
		if err != nil {
			return fetchResultMsg{err: err}
		}
		return fetchResultMsg{videos: videos, total: total}
	}
}

type downloadProgressMsg struct {
	Bvid    string
	Message string
}

func downloadCmd(bvid, workDir, ffmpegPath string, progressChan chan downloadProgressMsg) tea.Cmd {
	return func() tea.Msg {
		err := downloader.DownloadVideo(bvid, workDir, ffmpegPath, 0, downloader.DownloadPreference{}, func(msg string) {
			if progressChan != nil {
				progressChan <- downloadProgressMsg{Bvid: bvid, Message: msg}
			}
		})
		return downloadResultMsg{bvid: bvid, err: err}
	}
}

type searchUserResultMsg struct {
	users []crawler.BiliUserMeta
	err   error
}

func searchUserCmd(keyword string) tea.Cmd {
	return func() tea.Msg {
		users, err := crawler.SearchUsers(keyword)
		if err != nil {
			return searchUserResultMsg{err: err}
		}
		return searchUserResultMsg{users: users}
	}
}
