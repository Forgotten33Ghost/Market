--
-- PostgreSQL database dump
--

-- Dumped from database version 17.4
-- Dumped by pg_dump version 17.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: items; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.items (
    i_id smallint NOT NULL,
    i_available boolean DEFAULT true,
    i_name character varying(50) NOT NULL,
    i_price integer,
    i_gramm boolean DEFAULT false,
    i_URL character varying(100)
);


ALTER TABLE public.items OWNER TO admin;

--
-- Name: items_i_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.items_i_id_seq
    AS smallint
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.items_i_id_seq OWNER TO admin;

--
-- Name: items_i_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.items_i_id_seq OWNED BY public.items.i_id;

--
-- Name: items i_id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.items ALTER COLUMN i_id SET DEFAULT nextval('public.items_i_id_seq'::regclass);

--
-- Data for Name: items; Type: TABLE DATA; Schema: public; Owner: admin
--

-- Раскомментить для тестов
-- COPY public.items (i_id, i_available, i_name, i_price, i_gramm) FROM stdin;
-- 1	t	Сухари с изюмом	200	f
-- 2	t	Веселый сырник (с творогом)	400	f
-- \.

--
-- Name: items_i_id_seq; Type: SEQUENCE SET; Schema: public; Owner: admin
--

-- Раскомментить для тестов
-- SELECT pg_catalog.setval('public.items_i_id_seq', 2, true);

--
-- PostgreSQL database dump complete
--
