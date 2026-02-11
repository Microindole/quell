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
		videos, total, err := crawler.GetUserVideos(uid, page)
		if err != nil {
			return fetchResultMsg{err: err}
		}
		return fetchResultMsg{videos: videos, total: total}
	}
}

func downloadCmd(bvid, workDir, ffmpegPath string) tea.Cmd {
	return func() tea.Msg {
		err := downloader.DownloadVideo(bvid, workDir, ffmpegPath, nil)
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
