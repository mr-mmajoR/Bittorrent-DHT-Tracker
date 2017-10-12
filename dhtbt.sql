-- phpMyAdmin SQL Dump
-- version 4.5.4.1deb2ubuntu2
-- http://www.phpmyadmin.net
--
-- Host: localhost
-- Generation Time: Sep 27, 2017 at 09:39 AM
-- Server version: 5.7.19-0ubuntu0.16.04.1
-- PHP Version: 7.0.22-0ubuntu0.16.04.1

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `dhtbt`
--

-- --------------------------------------------------------

--
-- Table structure for table `files`
--

CREATE TABLE `files` (
  `infohash_id` int(11) NOT NULL,
  `path` varchar(250) NOT NULL,
  `length` bigint(40) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `infohash`
--

CREATE TABLE `infohash` (
  `id` int(11) NOT NULL,
  `infohash` varchar(40) NOT NULL,
  `name` varchar(250) NOT NULL,
  `length` bigint(40) NOT NULL,
  `files` tinyint(1) NOT NULL,
  `addeded` datetime NOT NULL,
  `updated` datetime NOT NULL,
  `cnt` int(11) NOT NULL DEFAULT '0',
  `textindex` mediumtext NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Indexes for dumped tables
--

--
-- Indexes for table `files`
--
ALTER TABLE `files`
  ADD KEY `path` (`path`) USING BTREE,
  ADD KEY `infohash_id` (`infohash_id`);

--
-- Indexes for table `infohash`
--
ALTER TABLE `infohash`
  ADD PRIMARY KEY (`id`),
  ADD KEY `name` (`name`),
  ADD KEY `cnt` (`cnt`),
  ADD KEY `updated` (`updated`),
  ADD KEY `files` (`files`);
ALTER TABLE `infohash` ADD FULLTEXT KEY `textindex` (`textindex`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `infohash`
--
ALTER TABLE `infohash`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=267277;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
